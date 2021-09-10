package main

import (
	"encoding/json"
	"fmt"
	yaml "gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Api struct {
	Uri    string
	Method string
}

type Apis struct {
	List      Api
	Member    Api
	Subscribe Api
}

type Miaomiao struct {
	Host    string
	Headers map[string]string
	Apis    Apis
	Params  map[string]interface{}
}
type Config struct {
	Tk        string `yaml:"tk"`
	StartTime string `yaml:"start_time"`
	Delay     int64  `yaml:"delay"`
}

func loadConfig() *Config {
	yamlFile, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		fmt.Println(err.Error())
	}
	c := &Config{}

	err = yaml.Unmarshal(yamlFile, c)

	if err != nil {
		log.Fatal(err.Error())
	}
	return c
}

func main() {
	c := loadConfig()
	m := &Miaomiao{
		Host: "https://miaomiao.scmttec.com",
		Headers: map[string]string{
			"tk":         c.Tk,
			"User-Agent": "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/53.0.2785.143 Safari/537.36 MicroMessenger/7.0.9.501 NetType/WIFI MiniProgramEnv/Windows WindowsWechat",
		},
		Apis: Apis{
			List:      Api{Uri: "/subscribe/subscribe/list.do?offset=0&limit=10&regionCode=5101", Method: "GET"},
			Member:    Api{Uri: "/subscribe/linkman/findByUserId.do", Method: "GET"},
			Subscribe: Api{Uri: "/subscribe/subscribe/subscribe.do", Method: "GET"},
		},
		Params: map[string]interface{}{
			"vaccineIndex": "1",
		},
	}

	m.members()
	m.hospitals()

	fmt.Printf("配置完成:[%+v]\r\n等待开始[%s]...\r\n", m.Params, c.StartTime)
	for {
		time.Sleep(time.Millisecond * 1000)
		if time.Now().Format("2006-01-02 15:04:05") > c.StartTime {
			break
		}
	}
	fmt.Println("开始预约......")
	var success bool
	for {
		go func() {
			result, err := m.subscribe()
			if err != nil {
				log.Printf("subscribe err: %s", err.Error())
			}
			if result {
				success = true
			}
			log.Printf("抢票失败\r\n")
		}()

		if success {
			log.Println("抢票成功")
			break
		}

		time.Sleep(time.Millisecond * time.Duration(c.Delay))
	}

	select {}
}

func (m *Miaomiao) hospitals() {
	resp, err := m.Request(m.Apis.List)

	if err != nil {
		log.Fatal("hospitals:", err)
	}

	if !resp.Ok {
		log.Fatalf("hospitals response error: %+v\r\n", resp)
	}

	if len(resp.Data.([]interface{})) == 0 {
		log.Fatal("该地区暂无秒杀信息")
	}

	hospitals := resp.Data.([]interface{})

	for i, hospital := range hospitals {
		if h, ok := hospital.(map[string]interface{}); ok {
			fmt.Printf("%d\t%s(%s)\r\n", i, h["name"], h["idCardNo"])
		}
	}

	var index int64
	fmt.Println("请输入接种医院index：")
	for {
		fmt.Scanln(&index)
		if hospital, ok := hospitals[index].(map[string]interface{}); ok {
			m.Params["seckillId"] = int(hospital["id"].(float64))
			fmt.Printf("已选择接种医院 id: %d %s\r\n", m.Params["seckillId"], hospital["name"])
			break
		}
		fmt.Println("输入信息不合法，请重新输入接种医院index：")
	}
}

func (m *Miaomiao) members() {
	resp, err := m.Request(m.Apis.Member)
	if err != nil {
		log.Fatal("members:", err)
	}

	if !resp.Ok {
		log.Fatalf("members response error: %+v\r\n", resp)
	}

	fmt.Println("index\t接种人")
	members := resp.Data.([]interface{})

	for i, member := range members {
		if mem, ok := member.(map[string]interface{}); ok {
			fmt.Printf("%d\t%s(%s)\r\n", i, mem["name"], mem["idCardNo"])
		}
	}
	var index int64
	fmt.Println("请输入接种人index：")
	for {
		fmt.Scanln(&index)
		if member, ok := members[index].(map[string]interface{}); ok {
			m.Params["linkmanId"] = int(member["id"].(float64))
			m.Params["idCardNo"] = member["idCardNo"]
			fmt.Printf("已选择接种人 id: %d %s(%s)\r\n", m.Params["linkmanId"], member["name"], m.Params["idCardNo"])
			break
		}
		fmt.Println("输入信息不合法，请重新输入接种人index：")
	}
}

func (m Miaomiao) subscribe() (bool, error) {
	//rand.Seed(time.Now().Unix())
	//num := rand.Intn(100)
	//if num > 90 {
	//	return true, nil
	//}
	response, err := m.Request(m.Apis.Subscribe)
	if err != nil {
		return false, err
	}
	if response.Code == "0" {
		return true, nil
	} else {
		log.Println("subscribe: ", response.Msg)
	}
	return false, nil
}

type Response struct {
	Code  string `json:"code"`
	Msg   string `json:"msg"`
	NotOk bool   `json:"notOk"`
	Ok    bool   `json:"ok"`
	Data  interface{}
}

func (m Miaomiao) Request(api Api) (*Response, error) {
	url := m.Host + api.Uri
	client := &http.Client{}
	request, err := http.NewRequest(api.Method, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	for key, value := range m.Headers {
		request.Header.Add(key, value)
	}

	response, _ := client.Do(request)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(response.Body)

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	//fmt.Printf("response body: %s\r\n", body)
	resp := &Response{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
