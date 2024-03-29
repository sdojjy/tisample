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
	"errors"
	//	"encoding/json"
	"fmt"
	//	"time"

	"github.com/luyomo/tisample/pkg/ctxt"
)

type CreateRouteTgw struct {
	pexecutor       *ctxt.Executor
	subClusterType  string
	subClusterTypes []string
}

// Execute implements the Task interface
func (c *CreateRouteTgw) Execute(ctx context.Context) error {
	clusterName := ctx.Value("clusterName").(string)
	clusterType := ctx.Value("clusterType").(string)

	sourceVpcInfo, err := getVPCInfo(*c.pexecutor, ctx, ResourceTag{clusterName: clusterName, clusterType: clusterType, subClusterType: c.subClusterType})
	if err != nil {
		return err
	}
	if sourceVpcInfo == nil {
		fmt.Printf("No source vpc info\n\n\n")
		return nil
	}
	fmt.Printf("The source vpc info is <%s> \n\n\n", (*sourceVpcInfo).CidrBlock)

	routeTable, err := getRouteTable(*c.pexecutor, ctx, clusterName, clusterType, c.subClusterType)
	if err != nil {
		return err
	}
	fmt.Printf("The route table is <%#v> \n\n\n", routeTable)

	transitGateway, err := getTransitGateway(*c.pexecutor, ctx, clusterName)
	if err != nil {
		return err
	}
	if transitGateway == nil {
		return errors.New("No transit gateway found")
	}

	fmt.Printf("The transit gateway is <%#v> \n\n\n", transitGateway)

	for _, targetSubClusterType := range c.subClusterTypes {
		fmt.Printf("The data is <%#v> \n\n\n", targetSubClusterType)
		//		vpcInfo, err := getVPC(local, ctx, clusterName, clusterType, targetSubClusterType)
		vpcInfo, err := getVPCInfo(*c.pexecutor, ctx, ResourceTag{clusterName: clusterName, clusterType: clusterType, subClusterType: targetSubClusterType})
		if err != nil {
			return err
		}
		if vpcInfo == nil {
			continue
		}
		fmt.Printf("The vpc info is <%s> \n\n\n", (*vpcInfo).CidrBlock)

		command := fmt.Sprintf("aws ec2 create-route --route-table-id %s --destination-cidr-block %s --transit-gateway-id %s", routeTable.RouteTableId, (*vpcInfo).CidrBlock, transitGateway.TransitGatewayId)
		fmt.Printf("The comamnd is <%s> \n\n\n", command)
		_, stderr, err := (*c.pexecutor).Execute(ctx, command, false)
		if err != nil {
			fmt.Printf("The error here is <%#v> \n\n\n", err)
			fmt.Printf("The error here is <%s> \n\n\n", string(stderr))
			return err
		}

		targetRouteTable, err := getRouteTable(*c.pexecutor, ctx, clusterName, clusterType, targetSubClusterType)
		if err != nil {
			return err
		}

		command = fmt.Sprintf("aws ec2 create-route --route-table-id %s --destination-cidr-block %s --transit-gateway-id %s", targetRouteTable.RouteTableId, (*sourceVpcInfo).CidrBlock, transitGateway.TransitGatewayId)

		_, stderr, err = (*c.pexecutor).Execute(ctx, command, false)
		if err != nil {
			fmt.Printf("The comamnd is <%s> \n\n\n", command)
			fmt.Printf("The error here is <%#v> \n\n\n", err)
			fmt.Printf("The error here is <%s> \n\n\n", string(stderr))
			return err
		}

	}

	return nil
}

// Rollback implements the Task interface
func (c *CreateRouteTgw) Rollback(ctx context.Context) error {
	return ErrUnsupportedRollback
}

// String implements the fmt.Stringer interface
func (c *CreateRouteTgw) String() string {
	return fmt.Sprintf("Echo: Creating route tgw ")
}
