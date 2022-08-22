/*
Copyright 2022 k0s authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package basic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/k0sproject/k0s/inttest/common"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CapitalHostnamesSuite struct {
	common.FootlooseSuite
}

func (s *CapitalHostnamesSuite) TestK0sGetsUp() {

	s.NoError(s.setHostname(s.ControllerNode(0), "k0s-CONTROLLER"))
	s.NoError(s.setHostname(s.WorkerNode(0), "k0s-WORKER"))

	s.NoError(s.InitController(0))

	token, err := s.GetJoinToken("worker")
	s.NoError(err)
	s.NoError(s.RunWorkersWithToken(token))

	kc, err := s.KubeClient(s.ControllerNode(0))
	if err != nil {
		s.FailNow("failed to obtain Kubernetes client", err)
	}

	err = s.WaitForNodeReady("k0s-worker", kc)
	s.NoError(err)

	pods, err := kc.CoreV1().Pods("kube-system").List(context.TODO(), v1.ListOptions{
		Limit: 100,
	})
	s.NoError(err)

	podCount := len(pods.Items)

	s.T().Logf("found %d pods in kube-system", podCount)
	s.Greater(podCount, 0, "expecting to see few pods in kube-system namespace")

	s.T().Log("waiting to see kube-router pods ready")
	s.NoError(common.WaitForKubeRouterReady(kc), "kube-router did not start")

	// Test that we get logs, it's a signal that konnectivity tunnels work
	s.T().Log("waiting to get logs from pods")
	s.Require().NoError(common.WaitForPodLogs(kc, "kube-system"))

	// Verify API that we get proper controller counter lease
	_, err = kc.CoordinationV1().Leases("kube-node-lease").Get(context.TODO(), "k0s-ctrl-k0s-controller", v1.GetOptions{})
	s.NoError(err)
}

func (s *CapitalHostnamesSuite) setHostname(node, hostname string) error {
	ssh, err := s.SSH(node)
	if err != nil {
		return err
	}
	defer ssh.Disconnect()

	_, err = ssh.ExecWithOutput("hostname " + hostname)
	return err
}

func TestCapitalHostnamesSuite(t *testing.T) {
	s := CapitalHostnamesSuite{
		common.FootlooseSuite{
			ControllerCount: 1,
			WorkerCount:     1,
		},
	}
	suite.Run(t, &s)
}