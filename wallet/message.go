package main

import (
	"encoding/json"
	"github.com/BerithFoundation/berith-chain/common"
	"github.com/BerithFoundation/berith-chain/rpc"
	"github.com/BerithFoundation/berith-chain/wallet/database"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bootstrap"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
	"reflect"
	"strings"
)


// handleMessages handles messages
func handleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "init":
		ch <- NodeMsg{
			t: "init",
			v: nil,
		}
		break
	case "callApi":
		var info map[string]interface{}
		err = json.Unmarshal(m.Payload, &info)
		if err != nil{
			payload = nil
			break
		}

		api := info["api"]
		args := info["args"].([]interface{})
		payload, err = callNodeApi(api, args...)
		break
	case "callDB":
		var info map[string]interface{}
		err = json.Unmarshal(m.Payload, &info)
		if err != nil{
			payload = nil
			break
		}
		api := info["api"]
		args := info["args"].([]interface{})
		payload , err = callDB(api , args...)
		break
	}
	return
}

func callNodeApi(api interface{}, args ...interface{}) (string, error)  {
	var result json.RawMessage
	p := make([]interface{}, 0)
	for _, item := range args{
		if item == nil {
			break
		}
		// 트랜잭션시
		if api.(string) == "berith_sendTransaction" {
			temp := reflect.ValueOf(item).Interface()
			itemMap:= temp.(map[string]interface{})
			argTemp := map[string]interface{}{
				"from" : reflect.ValueOf(itemMap["from"]).String(),
				"to" : reflect.ValueOf(itemMap["to"]).String(),
				"value" : reflect.ValueOf(itemMap["value"]).String(),
			}
			p = append(p, argTemp)
		}else if api.(string) == "berith_stake" || api.(string) == "berith_rewardToBalance"  || api.(string) == "berith_rewardToStake"{
			temp := reflect.ValueOf(item).Interface()
			itemMap:= temp.(map[string]interface{})
			argTemp := map[string]interface{}{
				"from" : reflect.ValueOf(itemMap["from"]).String(),
				"value" : reflect.ValueOf(itemMap["value"]).String(),
			}
			p = append(p, argTemp)
		}else{
			p = append(p , item)
		}
	}
	err := client.Call(&result, api.(string), p...)

	var val string
	switch err := err.(type) {
	case nil:
		if result == nil {

		} else {
			val = string(result)
			return val, err
		}
	case rpc.Error:
		return val, err
	default:
		return val, err
	}

	return val, err
}
func callDB ( api interface{}, args... interface{}) ( interface{}, error){
	key := make([]string, 0)
	for _, item := range args{
		 key  = append(key, item.(string))
	}
	acc ,err := callNodeApi("berith_coinbase", nil)
	acc = strings.ReplaceAll(acc , "\"","")
	if err != nil {
		astilog.Error(errors.Wrap(err, "insert error"))
	}
	switch api.(string) {
	case "selectContact" :
		contact := make(walletdb.Contact,0)
		err := WalletDB.Select([]byte(acc+"-contact"), &contact)
		if err != nil {
			return nil, err
		}
		return contact, nil

		break
	case "selectMember":
		var member walletdb.Member

		err := WalletDB.Select([]byte(acc+"-member"), &member)
		if err != nil {
			return nil, err
		}
		return member, nil
		break
	case "insertContact":
		contact := make(walletdb.Contact, 0)
		WalletDB.Select([]byte(acc), &contact)
		contact[common.HexToAddress(key[0])] = key[1]
		err := WalletDB.Insert([]byte(acc+"-contact") , contact)
		if err != nil {
			return nil, err
		}
		return  nil , nil
		break
	case "insertMember":
		member := walletdb.Member{
			Address: common.HexToAddress(acc),
			ID : key[0],
			Password: key[1],
		}
		member.PrivateKey[0] = 12
		err = WalletDB.Insert([]byte(acc+"-member") , member)
		if err != nil {
			return nil , err
		}
		break

	}

	return nil ,nil
}

