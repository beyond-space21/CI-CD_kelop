package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"bytes"
)

type PyParam struct {
	Mod string `json:"mod"`
	Arg []any  `json:"arg"`
}

func doPost(url string, body []byte) (map[string]interface{}, error) {

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		response := map[string]interface{}{"error": resp.StatusCode, "link": url, "payload": body}
		return response, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		fmt.Println(err)
		response := map[string]interface{}{"error": "error in unmarshalling","link":url,"payload":body}
		return response, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	return result, nil

}


func Pycess(det PyParam) (string,error) {

	jsonBytes, err := json.Marshal(det)
	if err != nil {
		panic(err)
	}

	response, err := doPost(PythonServer, jsonBytes)
	if err != nil {
		fmt.Println("doPost/process:", err)
		panic(err)
	}

	rtn := response["received"].(string)	
	if rtn[0] == '!'{
		return "", fmt.Errorf("%s", rtn)
	}
	return rtn,nil

}