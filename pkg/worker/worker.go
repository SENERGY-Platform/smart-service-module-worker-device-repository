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
	"context"
	"runtime/debug"
	"sync"

	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
)

func New(ctx context.Context, wg *sync.WaitGroup, config Config, libConfig configuration.Config, auth *auth.Auth, smartServiceRepo SmartServiceRepo) (*ProcessDeploymentStart, error) {
	return &ProcessDeploymentStart{config: config, libConfig: libConfig, auth: auth, smartServiceRepo: smartServiceRepo}, nil
}

type ProcessDeploymentStart struct {
	config           Config
	libConfig        configuration.Config
	auth             *auth.Auth
	smartServiceRepo SmartServiceRepo
}

type SmartServiceRepo interface {
	GetInstanceUser(ctx context.Context, instanceId string) (userId string, err error)
	UseModuleDeleteInfo(ctx context.Context, info model.ModuleDeleteInfo) error
	ListExistingModules(ctx context.Context, processInstanceId string, query model.ModulQuery) (result []model.SmartServiceModule, err error)
}

func (this *ProcessDeploymentStart) Do(ctx context.Context, task model.CamundaExternalTask) (modules []model.Module, outputs map[string]interface{}, err error) {
	userId, err := this.smartServiceRepo.GetInstanceUser(ctx, task.ProcessInstanceId)
	if err != nil {
		this.libConfig.GetLogger().ErrorContext(ctx, "unable to get instance user", "error", err)
		return modules, outputs, err
	}
	token, err := this.auth.ExchangeUserToken(userId)
	if err != nil {
		this.libConfig.GetLogger().ErrorContext(ctx, "unable to exchange user token", "error", err)
		return modules, outputs, err
	}

	deviceGroupDeviceIds, createDeviceGroup, err := this.getDeviceGroupDeviceIds(task)
	if err != nil {
		this.libConfig.GetLogger().ErrorContext(ctx, "error in ProcessDeploymentStart.Do", "error", err)
		return modules, outputs, err
	}

	outputs = map[string]interface{}{}

	name := this.getName(task)

	key := this.getModuleKey(task)

	if createDeviceGroup {
		module, returnData, err := this.handleDeviceGroupCommand(ctx, token, task, deviceGroupDeviceIds, name, key)
		if err != nil {
			return modules, returnData, err
		}
		modules = append(modules, module)
		for k, v := range returnData {
			outputs[k] = v
		}
	}

	return modules, outputs, err
}

func (this *ProcessDeploymentStart) Undo(ctx context.Context, modules []model.Module, reason error) {
	this.libConfig.GetLogger().DebugContext(ctx, "UNDO", "reason", reason)
	for _, module := range modules {
		if module.DeleteInfo != nil && !isUpdate(module) {
			err := this.smartServiceRepo.UseModuleDeleteInfo(ctx, *module.DeleteInfo)
			if err != nil {
				this.libConfig.GetLogger().ErrorContext(ctx, "error in Undo", "error", err, "stack", string(debug.Stack()))
			}
		}
	}
}

const ModuleUpdateVersionField = "module_update_version"

func isUpdate(module model.Module) bool {
	_, versionFieldExists := module.ModuleData[ModuleUpdateVersionField]
	return versionFieldExists
}

func setModuleUpdateVersion(module *model.Module) {
	version, versionFieldExists := module.ModuleData[ModuleUpdateVersionField]
	if !versionFieldExists {
		module.ModuleData[ModuleUpdateVersionField] = 1
	}
	versionNum, versionIsNum := version.(float64)
	if !versionIsNum {
		module.ModuleData[ModuleUpdateVersionField] = 1
	}
	module.ModuleData[ModuleUpdateVersionField] = versionNum + 1
}

func (this *ProcessDeploymentStart) getModuleId(task model.CamundaExternalTask, suffix string) string {
	return task.ProcessInstanceId + "." + task.Id + "." + suffix
}
