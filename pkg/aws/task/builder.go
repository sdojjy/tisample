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
	"crypto/tls"
	"fmt"
	"path/filepath"

	operator "github.com/luyomo/tisample/pkg/aws/operation"
	"github.com/luyomo/tisample/pkg/aws/spec"
	"github.com/luyomo/tisample/pkg/executor"

	"github.com/luyomo/tisample/pkg/crypto"
	"github.com/luyomo/tisample/pkg/meta"
	"github.com/luyomo/tisample/pkg/proxy"
)

// Builder is used to build TiUP task
type Builder struct {
	tasks []Task
}

// NewBuilder returns a *Builder instance
func NewBuilder() *Builder {
	return &Builder{}
}

// RootSSH appends a RootSSH task to the current task collection
func (b *Builder) RootSSH(
	host string, port int, user, password, keyFile, passphrase string, sshTimeout, exeTimeout uint64,
	proxyHost string, proxyPort int, proxyUser, proxyPassword, proxyKeyFile, proxyPassphrase string, proxySSHTimeout uint64,
	sshType, defaultSSHType executor.SSHType,
) *Builder {
	if sshType == "" {
		sshType = defaultSSHType
	}
	b.tasks = append(b.tasks, &RootSSH{
		host:            host,
		port:            port,
		user:            user,
		password:        password,
		keyFile:         keyFile,
		passphrase:      passphrase,
		timeout:         sshTimeout,
		exeTimeout:      exeTimeout,
		proxyHost:       proxyHost,
		proxyPort:       proxyPort,
		proxyUser:       proxyUser,
		proxyPassword:   proxyPassword,
		proxyKeyFile:    proxyKeyFile,
		proxyPassphrase: proxyPassphrase,
		proxyTimeout:    proxySSHTimeout,
		sshType:         sshType,
	})
	return b
}

// UserSSH append a UserSSH task to the current task collection
func (b *Builder) UserSSH(
	host string, port int, deployUser string, sshTimeout, exeTimeout uint64,
	proxyHost string, proxyPort int, proxyUser, proxyPassword, proxyKeyFile, proxyPassphrase string, proxySSHTimeout uint64,
	sshType, defaultSSHType executor.SSHType,
) *Builder {
	if sshType == "" {
		sshType = defaultSSHType
	}
	b.tasks = append(b.tasks, &UserSSH{
		host:            host,
		port:            port,
		deployUser:      deployUser,
		timeout:         sshTimeout,
		exeTimeout:      exeTimeout,
		proxyHost:       proxyHost,
		proxyPort:       proxyPort,
		proxyUser:       proxyUser,
		proxyPassword:   proxyPassword,
		proxyKeyFile:    proxyKeyFile,
		proxyPassphrase: proxyPassphrase,
		proxyTimeout:    proxySSHTimeout,
		sshType:         sshType,
	})
	return b
}

// Func append a func task.
func (b *Builder) Func(name string, fn func(ctx context.Context) error) *Builder {
	b.tasks = append(b.tasks, &Func{
		name: name,
		fn:   fn,
	})
	return b
}

// ClusterSSH init all UserSSH need for the cluster.
func (b *Builder) ClusterSSH(
	topo spec.Topology,
	deployUser string, sshTimeout, exeTimeout uint64,
	proxyHost string, proxyPort int, proxyUser, proxyPassword, proxyKeyFile, proxyPassphrase string, proxySSHTimeout uint64,
	sshType, defaultSSHType executor.SSHType,
) *Builder {
	if sshType == "" {
		sshType = defaultSSHType
	}
	var tasks []Task
	topo.IterInstance(func(inst spec.Instance) {
		tasks = append(tasks, &UserSSH{
			host:            inst.GetHost(),
			port:            inst.GetSSHPort(),
			deployUser:      deployUser,
			timeout:         sshTimeout,
			exeTimeout:      exeTimeout,
			proxyHost:       proxyHost,
			proxyPort:       proxyPort,
			proxyUser:       proxyUser,
			proxyPassword:   proxyPassword,
			proxyKeyFile:    proxyKeyFile,
			proxyPassphrase: proxyPassphrase,
			proxyTimeout:    proxySSHTimeout,
			sshType:         sshType,
		})
	})

	b.tasks = append(b.tasks, &Parallel{inner: tasks})

	return b
}

// UpdateMeta maintain the meta information
func (b *Builder) UpdateMeta(cluster string, metadata *spec.ClusterMeta, deletedNodeIds []string) *Builder {
	b.tasks = append(b.tasks, &UpdateMeta{
		cluster:        cluster,
		metadata:       metadata,
		deletedNodeIDs: deletedNodeIds,
	})
	return b
}

// UpdateTopology maintain the topology information
func (b *Builder) UpdateTopology(cluster, profile string, metadata *spec.ClusterMeta, deletedNodeIds []string) *Builder {
	b.tasks = append(b.tasks, &UpdateTopology{
		metadata:       metadata,
		cluster:        cluster,
		profileDir:     profile,
		deletedNodeIDs: deletedNodeIds,
		tcpProxy:       proxy.GetTCPProxy(),
	})
	return b
}

// CopyFile appends a CopyFile task to the current task collection
func (b *Builder) CopyFile(src, dst, server string, download bool, limit int) *Builder {
	b.tasks = append(b.tasks, &CopyFile{
		src:      src,
		dst:      dst,
		remote:   server,
		download: download,
		limit:    limit,
	})
	return b
}

// Download appends a Downloader task to the current task collection
func (b *Builder) Download(component, os, arch string, version string) *Builder {
	b.tasks = append(b.tasks, NewDownloader(component, os, arch, version))
	return b
}

// CopyComponent appends a CopyComponent task to the current task collection
func (b *Builder) CopyComponent(component, os, arch string,
	version string,
	srcPath, dstHost, dstDir string,
) *Builder {
	b.tasks = append(b.tasks, &CopyComponent{
		component: component,
		os:        os,
		arch:      arch,
		version:   version,
		srcPath:   srcPath,
		host:      dstHost,
		dstDir:    dstDir,
	})
	return b
}

// InstallPackage appends a InstallPackage task to the current task collection
func (b *Builder) InstallPackage(srcPath, dstHost, dstDir string) *Builder {
	b.tasks = append(b.tasks, &InstallPackage{
		srcPath: srcPath,
		host:    dstHost,
		dstDir:  dstDir,
	})
	return b
}

// BackupComponent appends a BackupComponent task to the current task collection
func (b *Builder) BackupComponent(component, fromVer string, host, deployDir string) *Builder {
	b.tasks = append(b.tasks, &BackupComponent{
		component: component,
		fromVer:   fromVer,
		host:      host,
		deployDir: deployDir,
	})
	return b
}

// InitConfig appends a CopyComponent task to the current task collection
func (b *Builder) InitConfig(clusterName, clusterVersion string, specManager *spec.SpecManager, inst spec.Instance, deployUser string, ignoreCheck bool, paths meta.DirPaths) *Builder {
	b.tasks = append(b.tasks, &InitConfig{
		specManager:    specManager,
		clusterName:    clusterName,
		clusterVersion: clusterVersion,
		instance:       inst,
		deployUser:     deployUser,
		ignoreCheck:    ignoreCheck,
		paths:          paths,
	})
	return b
}

// ScaleConfig generate temporary config on scaling
func (b *Builder) ScaleConfig(clusterName, clusterVersion string, specManager *spec.SpecManager, topo spec.Topology, inst spec.Instance, deployUser string, paths meta.DirPaths) *Builder {
	b.tasks = append(b.tasks, &ScaleConfig{
		specManager:    specManager,
		clusterName:    clusterName,
		clusterVersion: clusterVersion,
		base:           topo,
		instance:       inst,
		deployUser:     deployUser,
		paths:          paths,
	})
	return b
}

// MonitoredConfig appends a CopyComponent task to the current task collection
func (b *Builder) MonitoredConfig(name, comp, host string, globResCtl meta.ResourceControl, options *spec.MonitoredOptions, deployUser string, tlsEnabled bool, paths meta.DirPaths) *Builder {
	b.tasks = append(b.tasks, &MonitoredConfig{
		name:       name,
		component:  comp,
		host:       host,
		globResCtl: globResCtl,
		options:    options,
		deployUser: deployUser,
		tlsEnabled: tlsEnabled,
		paths:      paths,
	})
	return b
}

// SSHKeyGen appends a SSHKeyGen task to the current task collection
func (b *Builder) SSHKeyGen(keypath string) *Builder {
	b.tasks = append(b.tasks, &SSHKeyGen{
		keypath: keypath,
	})
	return b
}

// SSHKeySet appends a SSHKeySet task to the current task collection
func (b *Builder) SSHKeySet(privKeyPath, pubKeyPath string) *Builder {
	b.tasks = append(b.tasks, &SSHKeySet{
		privateKeyPath: privKeyPath,
		publicKeyPath:  pubKeyPath,
	})
	return b
}

// EnvInit appends a EnvInit task to the current task collection
func (b *Builder) EnvInit(host, deployUser string, userGroup string, skipCreateUser bool) *Builder {
	b.tasks = append(b.tasks, &EnvInit{
		host:           host,
		deployUser:     deployUser,
		userGroup:      userGroup,
		skipCreateUser: skipCreateUser,
	})
	return b
}

// ClusterOperate appends a cluster operation task.
// All the UserSSH needed must be init first.
func (b *Builder) ClusterOperate(
	spec *spec.Specification,
	op operator.Operation,
	options operator.Options,
	tlsCfg *tls.Config,
) *Builder {
	b.tasks = append(b.tasks, &ClusterOperate{
		spec:    spec,
		op:      op,
		options: options,
		tlsCfg:  tlsCfg,
	})

	return b
}

// Mkdir appends a Mkdir task to the current task collection
func (b *Builder) Mkdir(user, host string, dirs ...string) *Builder {
	b.tasks = append(b.tasks, &Mkdir{
		user: user,
		host: host,
		dirs: dirs,
	})
	return b
}

// Rmdir appends a Rmdir task to the current task collection
func (b *Builder) Rmdir(host string, dirs ...string) *Builder {
	b.tasks = append(b.tasks, &Rmdir{
		host: host,
		dirs: dirs,
	})
	return b
}

// Shell command on cluster host
func (b *Builder) Shell(host, command, cmdID string, sudo bool) *Builder {
	b.tasks = append(b.tasks, &Shell{
		host:    host,
		command: command,
		sudo:    sudo,
		cmdID:   cmdID,
	})
	return b
}

// SystemCtl run systemctl on host
func (b *Builder) SystemCtl(host, unit, action string, daemonReload bool) *Builder {
	b.tasks = append(b.tasks, &SystemCtl{
		host:         host,
		unit:         unit,
		action:       action,
		daemonReload: daemonReload,
	})
	return b
}

// Sysctl set a kernel parameter
func (b *Builder) Sysctl(host, key, val string) *Builder {
	b.tasks = append(b.tasks, &Sysctl{
		host: host,
		key:  key,
		val:  val,
	})
	return b
}

// Limit set a system limit
func (b *Builder) Limit(host, domain, limit, item, value string) *Builder {
	b.tasks = append(b.tasks, &Limit{
		host:   host,
		domain: domain,
		limit:  limit,
		item:   item,
		value:  value,
	})
	return b
}

// CheckSys checks system information of deploy server
func (b *Builder) CheckSys(host, dir, checkType string, topo *spec.Specification, opt *operator.CheckOptions) *Builder {
	b.tasks = append(b.tasks, &CheckSys{
		host:     host,
		topo:     topo,
		opt:      opt,
		checkDir: dir,
		check:    checkType,
	})
	return b
}

// DeploySpark deployes spark as dependency of TiSpark
func (b *Builder) DeploySpark(inst spec.Instance, sparkVersion, srcPath, deployDir string) *Builder {
	sparkSubPath := spec.ComponentSubDir(spec.ComponentSpark, sparkVersion)
	return b.CopyComponent(
		spec.ComponentSpark,
		inst.OS(),
		inst.Arch(),
		sparkVersion,
		srcPath,
		inst.GetHost(),
		deployDir,
	).Shell( // spark is under a subdir, move it to deploy dir
		inst.GetHost(),
		fmt.Sprintf(
			"cp -rf %[1]s %[2]s/ && cp -rf %[3]s/* %[2]s/ && rm -rf %[1]s %[3]s",
			filepath.Join(deployDir, "bin", sparkSubPath),
			deployDir,
			filepath.Join(deployDir, sparkSubPath),
		),
		"",
		false, // (not) sudo
	).CopyComponent(
		inst.ComponentName(),
		inst.OS(),
		inst.Arch(),
		"", // use the latest stable version
		srcPath,
		inst.GetHost(),
		deployDir,
	).Shell( // move tispark jar to correct path
		inst.GetHost(),
		fmt.Sprintf(
			"cp -f %[1]s/*.jar %[2]s/jars/ && rm -f %[1]s/*.jar",
			filepath.Join(deployDir, "bin"),
			deployDir,
		),
		"",
		false, // (not) sudo
	)
}

// TLSCert generates certificate for instance and transfers it to the server
func (b *Builder) TLSCert(host, comp, role string, port int, ca *crypto.CertificateAuthority, paths meta.DirPaths) *Builder {
	b.tasks = append(b.tasks, &TLSCert{
		host:  host,
		comp:  comp,
		role:  role,
		port:  port,
		ca:    ca,
		paths: paths,
	})
	return b
}

// Parallel appends a parallel task to the current task collection
func (b *Builder) Parallel(ignoreError bool, tasks ...Task) *Builder {
	if len(tasks) > 0 {
		b.tasks = append(b.tasks, &Parallel{ignoreError: ignoreError, inner: tasks})
	}
	return b
}

// Serial appends the tasks to the tail of queue
func (b *Builder) Serial(tasks ...Task) *Builder {
	if len(tasks) > 0 {
		b.tasks = append(b.tasks, tasks...)
	}
	return b
}

// Build returns a task that contains all tasks appended by previous operation
func (b *Builder) Build() Task {
	// Serial handles event internally. So the following 3 lines are commented out.
	// if len(b.tasks) == 1 {
	//  return b.tasks[0]
	// }
	return &Serial{inner: b.tasks}
}

// Step appends a new StepDisplay task, which will print single line progress for inner tasks.
func (b *Builder) Step(prefix string, inner Task) *Builder {
	b.Serial(newStepDisplay(prefix, inner))
	return b
}

// ParallelStep appends a new ParallelStepDisplay task, which will print multi line progress in parallel
// for inner tasks. Inner tasks must be a StepDisplay task.
func (b *Builder) ParallelStep(prefix string, ignoreError bool, tasks ...*StepDisplay) *Builder {
	b.tasks = append(b.tasks, newParallelStepDisplay(prefix, ignoreError, tasks...))
	return b
}

// BuildAsStep returns a task that is wrapped by a StepDisplay. The task will print single line progress.
func (b *Builder) BuildAsStep(prefix string) *StepDisplay {
	inner := b.Build()
	return newStepDisplay(prefix, inner)
}

// GcloudCreateInstance appends a GcloudCreateInstance task to the current task collection
func (b *Builder) Deploy(user, host string) *Builder {
	b.tasks = append(b.tasks, &Deploy{
		user: user,
		host: host,
	})
	return b
}

func (b *Builder) CreateVpc(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateVpc{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateNetwork(user, host, clusterName, clusterType, subClusterType string, isPrivate bool, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateNetwork{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
		isPrivate:      isPrivate,
	})
	return b
}

func (b *Builder) CreateRouteTable(user, host, clusterName, clusterType, subClusterType string, isPrivate bool, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateRouteTable{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
		isPrivate:      isPrivate,
	})
	return b
}

func (b *Builder) CreateSecurityGroup(user, host, clusterName, clusterType, subClusterType string, isPrivate bool, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateSecurityGroup{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
		isPrivate:      isPrivate,
	})
	return b
}

func (b *Builder) CreatePDNodes(user, host, clusterName, clusterType, subClusterType string, awsTopoConfigs *spec.AwsTopoConfigs, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreatePDNodes{
		user:           user,
		host:           host,
		awsTopoConfigs: awsTopoConfigs,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateTiDBNodes(user, host, clusterName, clusterType, subClusterType string, awsTopoConfigs *spec.AwsTopoConfigs, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateTiDBNodes{
		user:           user,
		host:           host,
		awsTopoConfigs: awsTopoConfigs,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateTiKVNodes(user, host, clusterName, clusterType, subClusterType string, awsTopoConfigs *spec.AwsTopoConfigs, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateTiKVNodes{
		user:           user,
		host:           host,
		awsTopoConfigs: awsTopoConfigs,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateDMNodes(user, host, clusterName, clusterType, subClusterType string, awsTopoConfigs *spec.AwsTopoConfigs, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateDMNodes{
		user:           user,
		host:           host,
		awsTopoConfigs: awsTopoConfigs,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateTiCDCNodes(user, host, clusterName, clusterType, subClusterType string, awsTopoConfigs *spec.AwsTopoConfigs, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateTiCDCNodes{
		user:           user,
		host:           host,
		awsTopoConfigs: awsTopoConfigs,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateWorkstation(user, host, clusterName, clusterType, subClusterType string, awsWSConfigs *spec.AwsWSConfigs, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateWorkstation{
		user:           user,
		host:           host,
		awsWSConfigs:   awsWSConfigs,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateInternetGateway(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateInternetGateway{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) AcceptVPCPeering(user, host, clusterName, clusterType string) *Builder {
	b.tasks = append(b.tasks, &AcceptVPCPeering{
		user:        user,
		host:        host,
		clusterName: clusterName,
		clusterType: clusterType,
	})
	return b
}

func (b *Builder) DeployTiDB(user, host, clusterName, clusterType, subClusterType string, awsWSConfigs *spec.AwsWSConfigs, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &DeployTiDB{
		user:           user,
		host:           host,
		awsWSConfigs:   awsWSConfigs,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateRouteTgw(user, host string, clusterName, clusterType, subClusterType string, subClusterTypes []string) *Builder {
	b.tasks = append(b.tasks, &CreateRouteTgw{
		user:            user,
		host:            host,
		clusterName:     clusterName,
		clusterType:     clusterType,
		subClusterType:  subClusterType,
		subClusterTypes: subClusterTypes,
	})
	return b
}

func (b *Builder) DestroyEC(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyEC{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroySecurityGroup(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroySecurityGroup{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyVpcPeering(user, host, clusterName, clusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyVpcPeering{
		user:        user,
		host:        host,
		clusterName: clusterName,
		clusterType: clusterType,
	})
	return b
}

func (b *Builder) DestroyNetwork(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyNetwork{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyRouteTable(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyRouteTable{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyInternetGateway(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyInternetGateway{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyVpc(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyVpc{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) CreateDBSubnetGroup(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateDBSubnetGroup{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateDBClusterParameterGroup(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateDBClusterParameterGroup{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateDBCluster(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateDBCluster{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateDBParameterGroup(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateDBParameterGroup{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateDBInstance(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateDBInstance{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) DestroyDBInstance(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyDBInstance{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyDBCluster(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyDBCluster{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyDBParameterGroup(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyDBParameterGroup{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyDBClusterParameterGroup(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyDBClusterParameterGroup{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyDBSubnetGroup(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyDBSubnetGroup{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) CreateMS(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateMS{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) DeployTiDBInstance(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &DeployTiDBInstance{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) DeployTiCDC(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &DeployTiCDC{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) MakeDBObjects(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &MakeDBObjects{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateDMSSourceEndpoint(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateDMSSourceEndpoint{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateDMSTargetEndpoint(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateDMSTargetEndpoint{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateDMSInstance(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateDMSInstance{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateDMSSubnetGroup(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateDMSSubnetGroup{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateDMSTask(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.tasks = append(b.tasks, &CreateDMSTask{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
		clusterInfo:    clusterInfo,
	})
	return b
}

func (b *Builder) CreateTransitGateway(user, host, clusterName, clusterType string) *Builder {
	b.tasks = append(b.tasks, &CreateTransitGateway{
		user:        user,
		host:        host,
		clusterName: clusterName,
		clusterType: clusterType,
	})
	return b
}

func (b *Builder) CreateTransitGatewayVpcAttachment(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &CreateTransitGatewayVpcAttachment{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyDMSInstance(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyDMSInstance{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyDMSTask(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyDMSTask{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyDMSEndpoints(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyDMSEndpoints{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyDMSSubnetGroup(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyDMSSubnetGroup{
		user:           user,
		host:           host,
		clusterName:    clusterName,
		clusterType:    clusterType,
		subClusterType: subClusterType,
	})
	return b
}

func (b *Builder) DestroyTransitGatewayVpcAttachment(user, host, clusterName, clusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyTransitGatewayVpcAttachment{
		user:        user,
		host:        host,
		clusterName: clusterName,
		clusterType: clusterType,
	})
	return b
}

func (b *Builder) DestroyTransitGateway(user, host, clusterName, clusterType string) *Builder {
	b.tasks = append(b.tasks, &DestroyTransitGateway{
		user:        user,
		host:        host,
		clusterName: clusterName,
		clusterType: clusterType,
	})
	return b
}

func (b *Builder) CreateBasicResource(user, host, clusterName, clusterType, subClusterType string, isPrivate bool, clusterInfo *ClusterInfo) *Builder {
	if isPrivate == true {
		b.CreateVpc(user, host, clusterName, clusterType, subClusterType, clusterInfo).
			CreateRouteTable(user, host, clusterName, clusterType, subClusterType, isPrivate, clusterInfo).
			CreateNetwork(user, host, clusterName, clusterType, subClusterType, isPrivate, clusterInfo).
			CreateSecurityGroup(user, host, clusterName, clusterType, subClusterType, isPrivate, clusterInfo)
	} else {
		b.CreateVpc(user, host, clusterName, clusterType, subClusterType, clusterInfo).
			CreateRouteTable(user, host, clusterName, clusterType, subClusterType, isPrivate, clusterInfo).
			CreateNetwork(user, host, clusterName, clusterType, subClusterType, isPrivate, clusterInfo).
			CreateSecurityGroup(user, host, clusterName, clusterType, subClusterType, isPrivate, clusterInfo).
			CreateInternetGateway(user, host, clusterName, clusterType, subClusterType, clusterInfo)
	}
	return b
}

func (b *Builder) CreateWorkstationCluster(user, host, clusterName, clusterType, subClusterType string, awsWSConfigs *spec.AwsWSConfigs, clusterInfo *ClusterInfo) *Builder {
	clusterInfo.cidr = awsWSConfigs.CIDR
	clusterInfo.keyFile = awsWSConfigs.KeyFile

	b.CreateBasicResource(user, host, clusterName, clusterType, subClusterType, false, clusterInfo).
		CreateWorkstation(user, host, clusterName, clusterType, subClusterType, awsWSConfigs, clusterInfo)

	return b
}

func (b *Builder) CreateTiDBCluster(user, host, clusterName, clusterType, subClusterType string, awsTopoConfigs *spec.AwsTopoConfigs, clusterInfo *ClusterInfo) *Builder {
	clusterInfo.cidr = awsTopoConfigs.General.CIDR

	b.CreateBasicResource(user, host, clusterName, clusterType, subClusterType, true, clusterInfo).
		//		CreateWorkstation(user, host, clusterName, clusterType, subClusterType, awsTopoConfigs, clusterInfo).
		CreatePDNodes(user, host, clusterName, clusterType, subClusterType, awsTopoConfigs, clusterInfo).
		CreateTiDBNodes(user, host, clusterName, clusterType, subClusterType, awsTopoConfigs, clusterInfo).
		CreateTiKVNodes(user, host, clusterName, clusterType, subClusterType, awsTopoConfigs, clusterInfo).
		CreateDMNodes(user, host, clusterName, clusterType, subClusterType, awsTopoConfigs, clusterInfo).
		CreateTiCDCNodes(user, host, clusterName, clusterType, subClusterType, awsTopoConfigs, clusterInfo)
		//		DeployTiDB(user, host, clusterName, clusterType, subClusterType, awsTopoConfigs, clusterInfo)

	return b
}

func (b *Builder) CreateAurora(user, host, clusterName, clusterType, subClusterType string, awsAuroraConfigs *spec.AwsAuroraConfigs, clusterInfo *ClusterInfo) *Builder {
	clusterInfo.cidr = awsAuroraConfigs.CIDR

	b.CreateBasicResource(user, host, clusterName, clusterType, subClusterType, true, clusterInfo).
		CreateDBSubnetGroup(user, host, clusterName, clusterType, subClusterType, clusterInfo).
		CreateDBClusterParameterGroup(user, host, clusterName, clusterType, subClusterType, clusterInfo).
		CreateDBCluster(user, host, clusterName, clusterType, subClusterType, clusterInfo).
		CreateDBParameterGroup(user, host, clusterName, clusterType, subClusterType, clusterInfo).
		CreateDBInstance(user, host, clusterName, clusterType, subClusterType, clusterInfo)
	return b
}

func (b *Builder) DestroyBasicResource(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.DestroyInternetGateway(user, host, clusterName, clusterType, subClusterType).
		DestroySecurityGroup(user, host, clusterName, clusterType, subClusterType).
		DestroyNetwork(user, host, clusterName, clusterType, subClusterType).
		DestroyRouteTable(user, host, clusterName, clusterType, subClusterType).
		DestroyVpc(user, host, clusterName, clusterType, subClusterType)

	return b
}

func (b *Builder) DestroyEC2Nodes(user, host, clusterName, clusterType, subClusterType string) *Builder {

	b.DestroyEC(user, "127.0.0.1", clusterName, clusterType, subClusterType).
		DestroyBasicResource(user, "127.0.0.1", clusterName, clusterType, subClusterType)

	return b
}

func (b *Builder) DestroyDMSService(user, host, clusterName, clusterType, subClusterType string) *Builder {
	b.DestroyDMSTask(user, "127.0.0.1", clusterName, clusterType, subClusterType).
		DestroyDMSInstance(user, "127.0.0.1", clusterName, clusterType, subClusterType).
		DestroyDMSEndpoints(user, "127.0.0.1", clusterName, clusterType, subClusterType).
		DestroyDMSSubnetGroup(user, "127.0.0.1", clusterName, clusterType, subClusterType).
		DestroyBasicResource(user, "127.0.0.1", clusterName, clusterType, subClusterType)

	return b
}

func (b *Builder) DestroyTransitGateways(user, host, clusterName, clusterType string) *Builder {

	b.DestroyTransitGatewayVpcAttachment(user, "127.0.0.1", clusterName, clusterType).
		DestroyTransitGateway(user, "127.0.0.1", clusterName, clusterType)

	return b
}

func (b *Builder) DestroyAurora(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.DestroyDBInstance(user, "127.0.0.1", clusterName, clusterType, subClusterType).
		DestroyDBCluster(user, "127.0.0.1", clusterName, clusterType, subClusterType).
		DestroyDBParameterGroup(user, "127.0.0.1", clusterName, clusterType, subClusterType).
		DestroyDBClusterParameterGroup(user, "127.0.0.1", clusterName, clusterType, subClusterType).
		DestroyDBSubnetGroup(user, "127.0.0.1", clusterName, clusterType, subClusterType).
		DestroyBasicResource(user, "127.0.0.1", clusterName, clusterType, subClusterType)

	return b
}

func (b *Builder) CreateSqlServer(user, host, clusterName, clusterType, subClusterType string, awsMSConfigs *spec.AwsMSConfigs, clusterInfo *ClusterInfo) *Builder {
	clusterInfo.cidr = awsMSConfigs.CIDR
	clusterInfo.keyName = awsMSConfigs.KeyName
	clusterInfo.instanceType = awsMSConfigs.InstanceType
	clusterInfo.imageId = awsMSConfigs.ImageId

	b.CreateBasicResource(user, host, clusterName, clusterType, subClusterType, true, clusterInfo).
		CreateMS(user, host, clusterName, clusterType, subClusterType, clusterInfo)

	return b
}

func (b *Builder) DestroySqlServer(user, host, clusterName, clusterType, subClusterType string, clusterInfo *ClusterInfo) *Builder {
	b.DestroyEC(user, "127.0.0.1", clusterName, clusterType, subClusterType).
		DestroyBasicResource(user, "127.0.0.1", clusterName, clusterType, subClusterType)

	return b
}

func (b *Builder) CreateDMSService(user, host, clusterName, clusterType, subClusterType string, awsDMSConfigs *spec.AwsDMSConfigs, clusterInfo *ClusterInfo) *Builder {
	clusterInfo.cidr = awsDMSConfigs.CIDR
	clusterInfo.instanceType = awsDMSConfigs.InstanceType
	b.CreateBasicResource(user, host, clusterName, clusterType, subClusterType, true, clusterInfo).
		CreateDMSSubnetGroup(user, host, clusterName, clusterType, subClusterType, clusterInfo).
		CreateDMSInstance(user, host, clusterName, clusterType, subClusterType, clusterInfo).
		CreateDMSSourceEndpoint(user, host, clusterName, clusterType, subClusterType, clusterInfo).
		CreateDMSTargetEndpoint(user, host, clusterName, clusterType, subClusterType, clusterInfo)

	return b
}

func (b *Builder) SysbenchTiCDC(user, host, identityFile, clusterName, clusterType string, tidbConnInfo operator.TiDBConnInfo) *Builder {
	b.tasks = append(b.tasks, &SysbenchTiCDC{
		user:         user,
		host:         host,
		identityFile: identityFile, // The identity file for workstation user. To improve better.
		clusterName:  clusterName,
		clusterType:  clusterType,
		tidbConnInfo: tidbConnInfo,
	})
	return b
}
