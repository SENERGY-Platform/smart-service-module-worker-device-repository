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
	"net/url"

	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
)

func (this *ProcessDeploymentStart) handleDeviceGroupCommand(token auth.Token, task model.CamundaExternalTask, deviceIds []string, name string, key *string) (module model.Module, outputs map[string]interface{}, err error) {
	if key != nil {
		return this.handleDeviceGroupCommandWithKey(token, task, deviceIds, name, *key)
	} else {
		return this.handleDeviceGroupCreate(token, task, deviceIds, name, []string{})
	}
}

const DeviceGroupIdOutputFieldName = "device_group_id"

func idToEventId(id string) string {
	return "permission_done_" + id
}

func (this *ProcessDeploymentStart) handleDeviceGroupCreate(token auth.Token, task model.CamundaExternalTask, deviceIds []string, name string, keys []string) (module model.Module, outputs map[string]interface{}, err error) {
	outputs = map[string]interface{}{}
	deviceGroupId, err := this.createDeviceGroup(token, task, deviceIds, name, this.getWaitSetting(task))
	if err != nil {
		this.libConfig.GetLogger().Error("error in handleDeviceGroupCreate", "error", err)
		return module, outputs, err
	}
	outputs["done_event"] = idToEventId(deviceGroupId)
	outputs[DeviceGroupIdOutputFieldName] = deviceGroupId
	iotOption := model.IotOption{
		DeviceGroupSelection: &model.DeviceGroupSelection{Id: deviceGroupId},
	}
	iotOptionJson, _ := json.Marshal(iotOption)
	outputs["device_group_iot_option"] = string(iotOptionJson)
	module = model.Module{
		Id:               this.getModuleId(task, "create_device_group"),
		ProcesInstanceId: task.ProcessInstanceId,
		SmartServiceModuleInit: model.SmartServiceModuleInit{
			DeleteInfo: &model.ModuleDeleteInfo{
				Url:    this.config.DeviceManagerUrl + "/device-groups/" + url.PathEscape(deviceGroupId),
				UserId: token.GetUserId(),
			},
			Keys:       keys,
			ModuleType: this.config.CreateDeviceGroupModuleType,
			ModuleData: outputs,
		},
	}
	return module, outputs, nil
}

func (this *ProcessDeploymentStart) handleDeviceGroupCommandWithKey(token auth.Token, task model.CamundaExternalTask, deviceIds []string, name string, key string) (module model.Module, outputs map[string]interface{}, err error) {
	module, exists, err := this.getExistingModule(task.ProcessInstanceId, key, this.config.CreateDeviceGroupModuleType)
	if !exists {
		return this.handleDeviceGroupCreate(token, task, deviceIds, name, []string{key})
	}
	setModuleUpdateVersion(&module)

	deviceGroupIdInterface, ok := module.ModuleData[DeviceGroupIdOutputFieldName]
	if !ok {
		this.libConfig.GetLogger().Warn("device-group-id output not found in module", "error", err, "module", fmt.Sprintf("%#v", module))
		return this.handleDeviceGroupCreate(token, task, deviceIds, name, []string{key})
	}
	deviceGroupId, ok := deviceGroupIdInterface.(string)
	if !ok {
		err = fmt.Errorf("module device-group-id output is not string: \n %#v", module)
		this.libConfig.GetLogger().Error("error in handleDeviceGroupCommandWithKey", "error", err)
		return module, outputs, err
	}

	outputs = module.ModuleData
	err = this.updateDeviceGroup(token, task, deviceIds, name, deviceGroupId, this.getWaitSetting(task))
	if err != nil {
		this.libConfig.GetLogger().Error("error in handleDeviceGroupCommandWithKey", "error", err)
		return module, outputs, err
	}

	return module, outputs, nil
}

func (this *ProcessDeploymentStart) getExistingModule(processInstanceId string, key string, moduleType string) (module model.Module, exists bool, err error) {
	existingModules, err := this.smartServiceRepo.ListExistingModules(processInstanceId, model.ModulQuery{
		KeyFilter:  &key,
		TypeFilter: &moduleType,
	})
	if err != nil {
		this.libConfig.GetLogger().Error("error in getExistingModule", "error", err)
		return module, false, err
	}
	this.libConfig.GetLogger().Debug("existing module request", "processInstanceId", processInstanceId, "key", key, "moduleType", moduleType, "existingModules", existingModules)
	if len(existingModules) == 0 {
		return module, false, nil
	}
	if len(existingModules) > 1 {
		this.libConfig.GetLogger().Warn("more than one existing module found", "processInstanceId", processInstanceId, "key", key, "moduleType", moduleType, "existingModules", existingModules)
	}
	module.SmartServiceModuleInit = existingModules[0].SmartServiceModuleInit
	module.ProcesInstanceId = processInstanceId
	module.Id = existingModules[0].Id
	return module, true, nil
}
