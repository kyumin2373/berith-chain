package bsrr

import (
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/BerithFoundation/berith-chain/berith/selection"
	"github.com/BerithFoundation/berith-chain/common"
	"github.com/BerithFoundation/berith-chain/params"
)

func TestGetMaxMiningCandidates(t *testing.T) {
	var c = &BSRR{
		config: &params.BSRRConfig{
			Period:       10,
			Epoch:        360,
			Rewards:      common.StringToBig("20000"),
			StakeMinimum: common.StringToBig("100000000000000000000000"),
			SlashRound:   1000,
			ForkFactor:   0.3,
		},
	}
	tests := []struct {
		holders  int
		expected int
	}{
		{0, 0},                     // no holders
		{1, 1},                     // only one holders
		{10, 3},                    // equals to 0 point
		{8, 2},                     // less than 0.5 point
		{9, 3},                     // greater than or equals 0.5 point
		{35000, selection.MAX_MINERS}, // greater than staking.MAX_MINERS
	}

	for i, test := range tests {
		result := c.getMaxMiningCandidates(test.holders)
		if result != test.expected {
			t.Errorf("test #%d: expected : %d but %d", i, test.expected, result)
		}
	}
}

func TestGetDelay(t *testing.T) {
	var c = &BSRR{
		config: &params.BSRRConfig{
			Period:       0,
			Epoch:        360,
			Rewards:      common.StringToBig("20000"),
			StakeMinimum: common.StringToBig("100000000000000000000000"),
			SlashRound:   1000,
			ForkFactor:   1.0,
		},
		rankGroup: &common.ArithmeticGroup{CommonDiff: 3},
	}

	tests := []struct {
		rank  int
		delay time.Duration
	}{
		{-1, time.Duration(0)},
		{0, time.Duration(0)},
		{1, time.Duration(0)},

		{2, 1 * groupDelay},
		{3, 1*groupDelay + 1*termDelay},
		{4, 1*groupDelay + 2*termDelay},

		{5, 2 * groupDelay},
		{6, 2*groupDelay + 1*termDelay},
		{7, 2*groupDelay + 2*termDelay},
	}

	for i, tt := range tests {
		result, _ := c.getDelay(tt.rank)
		if result != tt.delay {
			t.Errorf("test #%d: rank : %d expected : %d but %d", i, tt.rank, tt.delay, result)
		}
	}
}

// 임시 테스트 함수
func TestGetReward(t *testing.T) {
	sum := big.NewInt(0)
	for i := 0; i <= 630720000; i++ {
	//for i := 360; i <= 360; i++ {
		sum = new(big.Int).Add(sum, getRewardTemp(int64(i)))
	}

	fmt.Println("sum = ", sum)
}


// 100년 3153600000초
// 10초당 블록이 한개 생성된다면 100년 동안 생성되는 블록은 315360000
// 5초당 블록이 한개 생성된다면 100년 동안 생성되는 블록은 630720000
// 26은 시작 리워드
//
func getRewardTemp(number int64) *big.Int {
	// 특정 블록 이후로 보상을 지급
	if number < 360 {
		return big.NewInt(0)
	}

	//공식이 10초 단위 이기때문
	// d는 0.5 제네시스 블록의 설정에 따라 달라짐
	d := float64(5) / 10
	// 6300000 블록이면 n이 3150000
	n := float64(number) * d

	var z float64 = 0
	// n이 6300000 이하면 z = 5
	if n <= 3150000 {
		z = 5
	}

	re := big.NewInt(int64((26 - math.Round(n/7370000)*0.5 + z) * d))
	// 마이너스 reward는 없다.
	if re.Cmp(big.NewInt(0)) <= 0 {
		re = big.NewInt(0)
	}
	return re
}