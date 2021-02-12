package SSPanelAPI

import (
	"errors"
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/crossfw/Air-Universe/pkg/structures"
	regexp "github.com/dlclark/regexp2"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

/*
[url, port, alertId, isTLS, transportMode]   (.*?)(?=;)
path	(?<=path=).*(?=\|)|(?<=path=).*
host	(?<=host=).*(?=\|)|(?<=host=).*
*/

func String2Uint32(s string) (uint32, error) {
	t, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(t), err
}

func (node *NodeInfo) GetNodeInfo(cfg *structures.BaseConfig, idIndex uint32) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("get users from sspanel failed")
		}
	}()
	//nodeInfo = new(NodeInfo)
	client := &http.Client{Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/mod_mu/nodes/%v/info?key=%s", cfg.Panel.URL, cfg.Panel.NodeIDs[idIndex], cfg.Panel.Key), nil)
	if err != nil {
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	rtn, err := simplejson.NewJson(bodyText)
	if err != nil {
		return
	}

	node.RawInfo = rtn.Get("data").Get("server").MustString()
	node.Sort = uint32(rtn.Get("data").Get("sort").MustInt())
	node.Id = cfg.Panel.NodeIDs[idIndex]
	node.idIndex = idIndex
	node.SpeedLimit = uint32(rtn.Get("data").Get("node_speedlimit").MustInt())

	err = node.parseRawInfo()
	return
}

/*
[url, port, alertId, isTLS, transportMode]   (^|(?<=;))([^;]*)(?=;)
path	(?<=path=).*?(?=\|)|(?<=path=).*
host	(?<=host=).*?(?=\|)|(?<=host=).*
*/
func (node *NodeInfo) parseRawInfo() (err error) {
	reBasicInfos, _ := regexp.Compile("(^|(?<=;))([^;]*)(?=;)", 1)
	rePath, _ := regexp.Compile("(?<=path=).*?(?=\\|)|(?<=path=).*", 1)
	reHost, _ := regexp.Compile("(?<=host=).*?(?=\\|)|(?<=host=).*", 1)
	reInsidePort, _ := regexp.Compile("(?<=inside_port=).*?(?=\\|)|(?<=inside_port=).*", 1)
	reRelay, _ := regexp.Compile("\\|relay", 1)

	basicInfos, _ := reBasicInfos.FindStringMatch(node.RawInfo)
	var basicInfoArray []string
	for basicInfos != nil {
		basicInfoArray = append(basicInfoArray, basicInfos.String())
		basicInfos, _ = reBasicInfos.FindNextMatch(basicInfos)
	}
	mPath, _ := rePath.FindStringMatch(node.RawInfo)
	mHost, _ := reHost.FindStringMatch(node.RawInfo)
	mRelay, _ := reRelay.FindStringMatch(node.RawInfo)
	mInsidePort, _ := reInsidePort.FindStringMatch(node.RawInfo)
	//insidePort := mInsidePort
	if len(basicInfoArray) == 5 {
		node.Url = basicInfoArray[0]
		if mInsidePort == nil {
			node.ListenPort, _ = String2Uint32(basicInfoArray[1])
		} else {
			node.ListenPort, _ = String2Uint32(mInsidePort.String())
		}
		node.AlertID, _ = String2Uint32(basicInfoArray[2])

		node.TransportMode = basicInfoArray[3]

		if basicInfoArray[4] == "tls" {
			node.EnableTLS = true
		} else {
			node.EnableTLS = false
		}

	} else {
		err = errors.New("panel config missing params")
	}

	if mPath != nil {
		// First cheater is "\", remove it.
		node.Path = mPath.String()[1:]
	}
	if mRelay != nil {
		node.EnableProxyProtocol = true
	} else {
		node.EnableProxyProtocol = false
	}
	if mHost != nil {
		node.Host = mHost.String()
	}

	switch node.Sort {
	case 11:
		node.Protocol = "v2ray"
	case 12:
		node.Protocol = "v2ray"
		node.EnableProxyProtocol = true
	case 14:
		node.Protocol = "trojan"
	}

	return
}