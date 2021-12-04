// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package task

import (
	"context"
	"fmt"
	"github.com/luyomo/tisample/pkg/executor"
)

type TransitGateway struct {
	TransitGatewayId  string `json:"TransitGatewayId"`
	TransitGatewayArn string `json:"TransitGatewayArn`
	State             string `json:"State"`
}

type TransitGateways struct {
	TransitGateways []TransitGateway `json:"TransitGateways"`
}

type CreateTransitGateway struct {
	user        string
	host        string
	clusterName string
	clusterType string
}

//
// create-transit-gateway --description testtisample --tag-specifications ...
// Execute implements the Task interface
func (c *CreateTransitGateway) Execute(ctx context.Context) error {
	local, err := executor.New(executor.SSHTypeNone, false, executor.SSHConfig{Host: "127.0.0.1", User: c.user})

	transitGateway, err := getTransitGateway(local, ctx, c.clusterName)
	if err != nil {
		return err
	}
	if transitGateway != nil {
		return nil
	}

	command := fmt.Sprintf("aws ec2 create-transit-gateway --description %s --tag-specifications --tag-specifications \"ResourceType=transit-gateway,Tags=[{Key=Name,Value=%s},{Key=Cluster,Value=%s}]\"", c.clusterName, c.clusterName, c.clusterType)
	fmt.Printf("The comamnd is <%s> \n\n\n", command)
	stdout, stderr, err := local.Execute(ctx, command, false)
	if err != nil {
		fmt.Printf("The error here is <%#v> \n\n", err)
		fmt.Printf("----------\n\n")
		fmt.Printf("The error here is <%s> \n\n", string(stderr))
		return err
	}
	fmt.Printf("The result from create-transit-gateway <%s> \n\n\n", string(stdout))

	//var replicationInstanceRecord ReplicationInstanceRecord
	//if err = json.Unmarshal(stdout, &replicationInstanceRecord); err != nil {
	//	fmt.Printf("*** *** The error here is %#v \n\n", err)
	//	return nil
	//}
	//DMSInfo.ReplicationInstanceArn = replicationInstanceRecord.ReplicationInstance.ReplicationInstanceArn
	//for i := 1; i <= 50; i++ {
	//	command = fmt.Sprintf("aws dms describe-replication-instances")
	//	stdout, stderr, err := local.Execute(ctx, command, false)
	///	if err != nil {
	//		fmt.Printf("The error err here is <%#v> \n\n", err)
	//		fmt.Printf("----------\n\n")
	//		fmt.Printf("The error stderr here is <%s> \n\n", string(stderr))
	//		return nil
	//	} else {
	//		var replicationInstances ReplicationInstances
	//		if err = json.Unmarshal(stdout, &replicationInstances); err != nil {
	//			fmt.Printf("*** *** The error here is %#v \n\n", err)
	//			return nil
	//		}
	//		fmt.Printf("The db cluster is <%#v> \n\n\n", replicationInstances)
	//		for _, replicationInstance := range replicationInstances.ReplicationInstances {
	//			existsResource := ExistsDMSResource(c.clusterType, c.subClusterType, c.clusterName, replicationInstance.ReplicationInstanceArn, local, ctx)
	//			if existsResource == true {
	//				if replicationInstance.ReplicationInstanceStatus == "available" {
	//					return nil
	//				}
	//			}
	//		}
	//	}

	//	time.Sleep(30 * time.Second)
	//}

	return nil
}

// Rollback implements the Task interface
func (c *CreateTransitGateway) Rollback(ctx context.Context) error {
	return ErrUnsupportedRollback
}

// String implements the fmt.Stringer interface
func (c *CreateTransitGateway) String() string {
	return fmt.Sprintf("Echo: host=%s ", c.host)
}
