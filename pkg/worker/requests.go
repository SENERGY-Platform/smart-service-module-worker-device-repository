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

package worker

import (
	"bytes"
	"encoding/json"
	"errors"
	devicemodel "github.com/SENERGY-Platform/device-manager/lib/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"io"
	"net/http"
	"runtime/debug"
)

func (this *ProcessDeploymentStart) createDeviceGroup(token auth.Token, task model.CamundaExternalTask, ids []string, name string) (groupId string, err error) {
	if ids == nil {
		ids = []string{}
	}
	deviceGroup := devicemodel.DeviceGroup{
		Name:      name,
		Criteria:  nil,
		DeviceIds: ids,
		Attributes: []devicemodel.Attribute{
			{
				Key:    "platform/generated",
				Value:  "true",
				Origin: this.config.AttributeOrigin,
			},
			{
				Key:    "platform/smart_service_task",
				Value:  task.Id,
				Origin: this.config.AttributeOrigin,
			},
			{
				Key:    "platform/smart_service_instance",
				Value:  task.ProcessInstanceId,
				Origin: this.config.AttributeOrigin,
			},
			{
				Key:    "platform/smart_service_definition",
				Value:  task.ProcessDefinitionId,
				Origin: this.config.AttributeOrigin,
			},
		},
	}
	if len(ids) > 0 {
		deviceGroup.Criteria, err = this.getDeviceGroupCriteria(token, ids)
	}

	if err != nil {
		return groupId, err
	}
	payload := new(bytes.Buffer)
	err = json.NewEncoder(payload).Encode(deviceGroup)
	if err != nil {
		debug.PrintStack()
		return groupId, err
	}
	req, err := http.NewRequest("POST", this.config.DeviceManagerUrl+"/device-groups", payload)
	if err != nil {
		debug.PrintStack()
		return groupId, err
	}
	req.Header.Set("Authorization", token.Jwt())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		debug.PrintStack()
		return groupId, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		err = errors.New(string(temp))
		debug.PrintStack()
		return groupId, err
	}
	err = json.NewDecoder(resp.Body).Decode(&deviceGroup)
	groupId = deviceGroup.Id
	return
}

func (this *ProcessDeploymentStart) getDeviceGroupCriteria(token auth.Token, ids []string) (result []devicemodel.DeviceGroupFilterCriteria, err error) {
	payload := new(bytes.Buffer)
	err = json.NewEncoder(payload).Encode(ids)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	req, err := http.NewRequest("POST", this.config.DeviceSelectionUrl+"/device-group-helper", payload)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	req.Header.Set("Authorization", token.Jwt())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		err = errors.New(string(temp))
		debug.PrintStack()
		return result, err
	}
	temp := DeviceGroupHelperResult{}
	err = json.NewDecoder(resp.Body).Decode(&temp)
	result = temp.Criteria
	return
}

type DeviceGroupHelperResult struct {
	Criteria []devicemodel.DeviceGroupFilterCriteria `json:"criteria"`
}
