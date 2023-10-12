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
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-module-worker-device-repository/pkg/kafka"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/camunda"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

func StartDoneEventHandling(ctx context.Context, wg *sync.WaitGroup, config Config, libConfig configuration.Config) error {
	if config.KafkaUrl != "" && config.KafkaUrl != "-" {
		return kafka.NewConsumer(ctx, wg, config.KafkaUrl, config.KafkaConsumerGroup, config.PermissionsDoneTopic, func(delivery []byte) error {
			msg := DoneNotification{}
			err := json.Unmarshal(delivery, &msg)
			if err != nil {
				log.Println("ERROR: unable to interpret kafka msg:", err)
				debug.PrintStack()
				return nil //ignore  message
			}
			if msg.Command == "PUT" && msg.Handler != "github.com/SENERGY-Platform/permission-search" {
				eventId := idToEventId(msg.ResourceKind + "_" + msg.ResourceId)
				err = camunda.SendEventTrigger(libConfig, eventId, nil)
				if err != nil {
					log.Println("ERROR: unable to send event trigger:", err)
					debug.PrintStack()
					return err
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
			return nil
		})
	}
	return nil
}

func idToEventId(id string) string {
	return "permission_done_" + id
}

type DoneNotification struct {
	ResourceKind string `json:"resource_kind"`
	ResourceId   string `json:"resource_id"`
	Handler      string `json:"handler"` // == github.com/SENERGY-Platform/permission-search
	Command      string `json:"command"` // PUT | DELETE | RIGHTS
}
