/*
 * Copyright (c) 2022 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mocks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime/debug"
	"sync"
)

type HttpService struct {
	requestsLog   []Request
	mux           sync.Mutex
	ResponseIndex map[string]map[string]int        //method -> path = request-count
	Response      map[string]map[string][]Response //method -> path -> request-count = response
}

type Request struct {
	Method   string `json:"method"`
	Endpoint string `json:"endpoint"`
	Message  string `json:"message"`
}

type Response struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

func (this *HttpService) ensureMapInit(method string) {
	if this.Response == nil {
		this.Response = map[string]map[string][]Response{}
	}
	if this.Response[method] == nil {
		this.Response[method] = map[string][]Response{}
	}
	if this.ResponseIndex == nil {
		this.ResponseIndex = map[string]map[string]int{}
	}
	if this.ResponseIndex[method] == nil {
		this.ResponseIndex[method] = map[string]int{}
	}
}

func (this *HttpService) SetResponse(method string, path string, responses []Response) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.ensureMapInit(method)
	this.Response[method][path] = responses
	this.ResponseIndex[method][path] = 0
}

func (this *HttpService) getResponse(request *http.Request) (Response, bool) {
	this.mux.Lock()
	defer this.mux.Unlock()
	method := request.Method
	path := request.URL.Path
	this.ensureMapInit(method)
	index, ok := this.ResponseIndex[method][path]
	if !ok {
		return Response{}, false
	}
	if responses, ok := this.Response[method][path]; ok && len(responses) > index {
		this.ResponseIndex[method][path] = index + 1
		return this.Response[method][path][index], true
	}
	return Response{}, false
}

func (this *HttpService) PopRequestLog() []Request {
	this.mux.Lock()
	defer this.mux.Unlock()
	result := this.requestsLog
	this.requestsLog = []Request{}
	return result
}

func (this *HttpService) GetRequestLog() []Request {
	this.mux.Lock()
	defer this.mux.Unlock()
	result := this.requestsLog
	return result
}

func (this *HttpService) logRequest(request *http.Request) {
	this.mux.Lock()
	defer this.mux.Unlock()
	temp, _ := io.ReadAll(request.Body)
	query := request.URL.RawQuery
	if query != "" {
		query = "?" + query
	}
	this.requestsLog = append(this.requestsLog, Request{
		Method:   request.Method,
		Endpoint: request.URL.Path + query,
		Message:  string(temp),
	})
}

func (this *HttpService) logRequestWithMessage(request *http.Request, m interface{}) {
	this.mux.Lock()
	defer this.mux.Unlock()
	message, ok := m.(string)
	if !ok {
		temp, _ := json.Marshal(m)
		message = string(temp)
	}
	query := request.URL.RawQuery
	if query != "" {
		query = "?" + query
	}
	this.requestsLog = append(this.requestsLog, Request{
		Method:   request.Method,
		Endpoint: request.URL.Path + query,
		Message:  message,
	})
}

func (this *HttpService) Start(ctx context.Context, wg *sync.WaitGroup) (url string) {
	server := httptest.NewServer(this.getRouter())
	wg.Add(1)
	go func() {
		<-ctx.Done()
		server.Close()
		wg.Done()
	}()
	return server.URL
}

func (this *HttpService) getRouter() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		this.logRequest(request)
		resp, ok := this.getResponse(request)
		if !ok {
			http.Error(writer, "mock: no response stored", 500)
			debug.PrintStack()
			return
		}
		writer.WriteHeader(resp.StatusCode)
		_, err := writer.Write([]byte(resp.Message))
		if err != nil {
			log.Println("ERROR:", err)
			debug.PrintStack()
		}
		return
	})
}

func (this *HttpService) CheckExpectedRequests(expectedRequests []Request) error {
	actualEngineRequests := this.GetRequestLog()
	if !reflect.DeepEqual(expectedRequests, actualEngineRequests) {
		a, _ := json.Marshal(actualEngineRequests)
		e, _ := json.Marshal(expectedRequests)
		return fmt.Errorf("\n %v \n %v", string(a), string(e))
	}
	return nil
}

func (this *HttpService) CheckExpectedRequestsFromFileLocation(fileLocation string) error {
	fileContent, err := os.ReadFile(fileLocation)
	if err != nil {
		return err
	}
	expectedRequests := []Request{}
	err = json.Unmarshal(fileContent, &expectedRequests)
	if err != nil {
		return err
	}
	return this.CheckExpectedRequests(expectedRequests)
}

func (this *HttpService) SetResponses(responsesMap map[string]map[string][]Response) {
	for method, paths := range responsesMap {
		for path, responses := range paths {
			this.SetResponse(method, path, responses)
		}
	}
}

func (this *HttpService) SetResponsesFromFile(fileLocation string) error {
	fileContent, err := os.ReadFile(fileLocation)
	if err != nil {
		return err
	}
	responses := map[string]map[string][]Response{}
	err = json.Unmarshal(fileContent, &responses)
	if err != nil {
		return err
	}
	this.SetResponses(responses)
	return nil
}
