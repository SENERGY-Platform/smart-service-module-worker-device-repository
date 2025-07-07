/*
 * Copyright (c) 2023 InfAI (CC SES)
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
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/camunda"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"log"
	"runtime/debug"
	"time"
)

func triggerDoneEvent(libConfig configuration.Config, resourceId string) {
	eventId := idToEventId(resourceId)
	err := camunda.SendEventTrigger(libConfig, eventId, nil)
	if err == nil {
		return
	}
	go func() {
		time.Sleep(5 * time.Second)
		err = camunda.SendEventTrigger(libConfig, eventId, nil)
		if err != nil {
			log.Println("ERROR: unable to send event trigger:", err)
			debug.PrintStack()
		}
	}()
}

func idToEventId(id string) string {
	return "permission_done_" + id
}
