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
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"log"
	"net/url"
	"runtime/debug"
)

func New(config Config, libConfig configuration.Config, auth *auth.Auth, smartServiceRepo SmartServiceRepo) *ProcessDeploymentStart {
	return &ProcessDeploymentStart{config: config, libConfig: libConfig, auth: auth, smartServiceRepo: smartServiceRepo}
}

type ProcessDeploymentStart struct {
	config           Config
	libConfig        configuration.Config
	auth             *auth.Auth
	smartServiceRepo SmartServiceRepo
}

type SmartServiceRepo interface {
	GetInstanceUser(instanceId string) (userId string, err error)
	UseModuleDeleteInfo(info model.ModuleDeleteInfo) error
}

func (this *ProcessDeploymentStart) Do(task model.CamundaExternalTask) (modules []model.Module, outputs map[string]interface{}, err error) {
	userId, err := this.smartServiceRepo.GetInstanceUser(task.ProcessInstanceId)
	if err != nil {
		log.Println("ERROR: unable to get instance user", err)
		return modules, outputs, err
	}
	token, err := this.auth.ExchangeUserToken(userId)
	if err != nil {
		log.Println("ERROR: unable to exchange user token", err)
		return modules, outputs, err
	}

	deviceGroupDeviceIds, createDeviceGroup, err := this.getDeviceGroupDeviceIds(task)
	if err != nil {
		log.Println("ERROR:", err)
		return modules, outputs, err
	}

	moduleData := map[string]interface{}{}

	name := this.getName(task)

	if createDeviceGroup {
		deviceGroupId, err := this.createDeviceGroup(token, task, deviceGroupDeviceIds, name)
		if err != nil {
			log.Println("ERROR:", err)
			return modules, outputs, err
		}
		moduleData["device_group_id"] = deviceGroupId
		iotOption := model.IotOption{
			DeviceGroupSelection: &model.DeviceGroupSelection{Id: deviceGroupId},
		}
		iotOptionJson, _ := json.Marshal(iotOption)
		moduleData["device_group_iot_option"] = string(iotOptionJson)
		modules = append(modules, model.Module{
			Id:               this.getModuleId(task, "create_device_group"),
			ProcesInstanceId: task.ProcessInstanceId,
			SmartServiceModuleInit: model.SmartServiceModuleInit{
				DeleteInfo: &model.ModuleDeleteInfo{
					Url:    this.config.DeviceManagerUrl + "/device-groups/" + url.PathEscape(deviceGroupId),
					UserId: userId,
				},
				ModuleType: this.config.CreateDeviceGroupModuleType,
				ModuleData: moduleData,
			},
		})
	}

	return modules, moduleData, err
}

func (this *ProcessDeploymentStart) Undo(modules []model.Module, reason error) {
	log.Println("UNDO:", reason)
	for _, module := range modules {
		if module.DeleteInfo != nil {
			err := this.smartServiceRepo.UseModuleDeleteInfo(*module.DeleteInfo)
			if err != nil {
				log.Println("ERROR:", err)
				debug.PrintStack()
			}
		}
	}
}

func (this *ProcessDeploymentStart) getModuleId(task model.CamundaExternalTask, suffix string) string {
	return task.ProcessInstanceId + "." + task.Id + "." + suffix
}
