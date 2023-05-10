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
	"encoding/json"
	"fmt"
	devicemodel "github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"strings"
)

const DeviceIdPrefix = devicemodel.URN_PREFIX + "device:"

func (this *ProcessDeploymentStart) getName(task model.CamundaExternalTask) string {
	defaultName := this.config.DefaultNamePrefix + task.ProcessInstanceId + "." + task.Id
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"name"]
	if !ok {
		return defaultName
	}
	result, ok := variable.Value.(string)
	if !ok || result == "" {
		return defaultName
	}
	return result
}

// if no key is set: return nil
func (this *ProcessDeploymentStart) getModuleKey(task model.CamundaExternalTask) (key *string) {
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"key"]
	if !ok {
		return nil
	}
	result, ok := variable.Value.(string)
	if ok {
		return &result
	}
	return nil
}

func (this *ProcessDeploymentStart) getDeviceGroupDeviceIds(task model.CamundaExternalTask) (deviceIds []string, used bool, err error) {
	key := this.config.WorkerParamPrefix + "create_device_group"
	variable, ok := task.Variables[key]
	if !ok {
		return nil, false, nil
	}
	list := []interface{}{}
	switch temp := variable.Value.(type) {
	case string:
		err = json.Unmarshal([]byte(temp), &list)
		if err != nil {
			return nil, true, fmt.Errorf("unable to unmarshal value of %v to []interface{}: %w", key, err)
		}
	case []string:
		for _, str := range temp {
			list = append(list, str)
		}
	case []interface{}:
		list = temp
	default:
		return nil, false, fmt.Errorf("unable to interpret value of %v (%#v)", key, variable.Value)
	}

	for _, element := range list {
		var id string
		if str, isStr := element.(string); isStr {
			id, ok = extractDeviceIdFromString(str)
		} else {
			temp, _ := json.Marshal(element)
			id, ok = extractDeviceIdFromString(string(temp))
		}
		if ok {
			deviceIds = append(deviceIds, id)
		}
	}
	return deviceIds, true, nil
}

func extractDeviceIdFromString(str string) (id string, ok bool) {
	if strings.HasPrefix(str, DeviceIdPrefix) {
		return str, true
	}
	if strings.HasPrefix(str, devicemodel.URN_PREFIX) {
		return "", false
	}
	iotOption := model.IotOption{}
	err := json.Unmarshal([]byte(str), &iotOption)
	if err == nil && iotOption.DeviceSelection != nil {
		return iotOption.DeviceSelection.DeviceId, true
	}
	return "", false
}
