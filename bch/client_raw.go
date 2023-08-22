package bch

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	testMempoolAcceptReqFmt = `{"jsonrpc": "1.0", "id":"relayer", "method": "testmempoolaccept", "params": [["%s"]] }`
)

type JsonRpcResult struct {
	Id     string          `json:"id"`
	Error  *JsonRpcError   `json:"error"`
	Result json.RawMessage `json:"result"`
}
type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// https://docs.bitcoincashnode.org/doc/json-rpc/testmempoolaccept/
type TestMempoolAcceptResult struct {
	Txid         string `json:"txid"`
	Allowed      bool   `json:"allowed"`
	RejectReason string `json:"reject-reason"`
}

func testMempoolAccept(rpcUrl, user, pass string, rawTx []byte) (bool, error) {
	req := fmt.Sprintf(testMempoolAcceptReqFmt, hex.EncodeToString(rawTx))
	resp, err := sendRequest(rpcUrl, user, pass, req)
	if err != nil {
		return false, fmt.Errorf("failed to send resquest: %w", err)
	}

	var jsonRpcResult JsonRpcResult
	err = json.Unmarshal(resp, &jsonRpcResult)
	if err != nil {
		return false, fmt.Errorf("failed to unmarsal JSON RPC result: %w", err)
	}

	if jsonRpcResult.Error != nil && jsonRpcResult.Error.Code != 0 {
		return false, fmt.Errorf("error code: %d, error message: %s",
			jsonRpcResult.Error.Code, jsonRpcResult.Error.Message)
	}

	var testMempoolAcceptResult []TestMempoolAcceptResult
	err = json.Unmarshal(jsonRpcResult.Result, &testMempoolAcceptResult)
	if err != nil {
		return false, fmt.Errorf("failed to unmarsal TestMempoolAcceptResults: %w", err)
	}

	if n := len(testMempoolAcceptResult); n != 1 {
		return false, fmt.Errorf("invalid TestMempoolAcceptResults count: %d", n)
	}
	if !testMempoolAcceptResult[0].Allowed {
		return false, fmt.Errorf("tx rejected: %s", testMempoolAcceptResult[0].RejectReason)
	}

	return true, nil
}

func sendRequest(host, user, pass, reqStr string) ([]byte, error) {
	body := strings.NewReader(reqStr)
	req, err := http.NewRequest("POST", host, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pass)
	req.Header.Set("Content-Type", "text/plain;")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	return respData, nil
}
