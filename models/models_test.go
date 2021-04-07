package models

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	controllerClient "github.com/veritone/realtime/modules/controller/client"
	"testing"
	"time"
)

func TestGetWorkRequestToMessageAndBack(t *testing.T) {
	registerTypes()
	p := GetWorkRequest{
		Name:         "name1",
		ID:           "id1",
		TTL:          1000,
		TimestampUTC: GetCurrentTimeEpochMs(),
	}

	bArr, err := SerializeToBytesForTransport(&p)
	pa2, err := ByteArrayToAType(bArr)
	assert.Nil(t, err)
	assert.NotNil(t, pa2)
	p2, ok := pa2.(*GetWorkRequest)
	assert.True(t, ok)
	assert.NotNil(t, p2)
	t.Logf("p2 = %s, err =%v", ToString(p2), err)
	assert.Equal(t, p.Name, p2.Name)
	assert.Equal(t, p.ID, p2.ID)
	assert.Equal(t, p.TimestampUTC, p2.TimestampUTC)
	assert.Equal(t, p.TTL, p2.TTL)

	p3, ok := pa2.(*GetWorkResponse)
	assert.False(t, ok)
	assert.Nil(t, p3)
}

func TestGetWorkResponseToMessageAndBack(t *testing.T) {
	p := GetWorkResponse{
		Name:         "name1",
		ID:           "id1",
		WorkItem:     "workItem1",
		TimestampUTC: GetCurrentTimeEpochMs(),
	}

	bArr, err := SerializeToBytesForTransport(&p)
	pa2, err := ByteArrayToAType(bArr)
	assert.Nil(t, err)
	assert.NotNil(t, pa2)
	p2, ok := pa2.(*GetWorkResponse)
	assert.True(t, ok)
	assert.NotNil(t, p2)
	t.Logf("p2 = %s, err =%v", ToString(p2), err)
	assert.Equal(t, p.Name, p2.Name)
	assert.Equal(t, p.ID, p2.ID)
	assert.Equal(t, p.TimestampUTC, p2.TimestampUTC)
	assert.Equal(t, p.WorkItem, p2.WorkItem)

	p3, ok := pa2.(*GetWorkRequest)
	assert.False(t, ok)
	assert.Nil(t, p3)
}

// go test -v -run="TestGetControllerClientStructs"
func TestGetControllerClientStructs(t *testing.T) {
	p := controllerClient.EngineInstanceWorkRequest{
		ContainerStatus: controllerClient.ContainerStatus{
			CPUUtilization: 95.5,
			ContainerID:    "containerID1",
		},
		EngineInstanceID:        "EI1",
		EngineToolkitVersion:    "v1",
		HostAction:              "haction",
		HostID:                  "host1",
		RequestWorkForEngineIds: []string{"Engine1"},
		TaskStatuses: []controllerClient.TaskStatusDetail{
			controllerClient.TaskStatusDetail{
				CreateJobsAction: []controllerClient.CreateJobsAction{
					controllerClient.CreateJobsAction{BaseInternalJobID: "job1",
						CreateTDODetail: controllerClient.CreateTdoDetail{
							Description:   "create1",
							IsPublic:      true,
							Name:          "name1",
							StartDateTime: time.Now(),
							StopDateTime:  time.Now().Add(5 * time.Minute),
						}},
				},
				EngineID:      "Engine1",
				ErrorCount:    5,
				ErrorDetails:  "ErrorDetails1",
				FailureReason: "Failur1",
				Inputs: []controllerClient.TaskIoStatus{
					controllerClient.TaskIoStatus{
						Id:                  "io1",
						ProcessedCount:      5,
						ProcessedTotalCount: 10,
					},
				},
				InternalJobID:  "IJ1",
				InternalTaskID: "IT1",
				Outputs:        nil,
				PriorTimestamp: 0,
				ProcessedStats: controllerClient.TaskProcessedStats{
					ProcessedBytes:           100,
					ProcessedCPUMilliseconds: 5000,
					ProcessedCPUSeconds:      5,
					ProcessedChunks:          5,
					ProcessedMediaSeconds:    3,
				},
				RetryCount: 0,
				TaskAction: "",
				TaskOutput: map[string]interface{}{
					"key1":       "value1",
					"timestamp1": time.Now().Format(time.RFC3339), // time is a bit iffy as well as float vs. int will need to use string
				},
			},
		},
		WorkRequestDetails: "WorkRequestDetail1",
		WorkRequestID:      "WorkRequestID1",
		WorkRequestStatus:  "Running",
	}

	bArr, err := SerializeToBytesForTransport(&p)
	assert.Nil(t, err)
	pa2, err := ByteArrayToAType(bArr)
	assert.Nil(t, err)
	assert.NotNil(t, pa2)
	p2, ok := pa2.(*controllerClient.EngineInstanceWorkRequest)
	assert.True(t, ok)
	assert.NotNil(t, p2)

	t.Logf("p2 = %s, err =%v", ToString(p2), err)

	assert.Equal(t, p.EngineInstanceID, p2.EngineInstanceID)
	assert.Equal(t, p.ContainerStatus.CPUUtilization, p2.ContainerStatus.CPUUtilization)
	assert.Equal(t, p.ContainerStatus.ContainerID, p2.ContainerStatus.ContainerID)
	assert.Equal(t, p.RequestWorkForEngineIds, p2.RequestWorkForEngineIds)
	b, where := taskStatusesEqual(p.TaskStatuses, p2.TaskStatuses)
	if !b {
		t.Logf("Failed taskStatuses in %s", where)
	}
	assert.True(t, b)

}

func taskStatusesEqual(ta1, ta2 []controllerClient.TaskStatusDetail) (bool, string) {

	if len(ta1) != len(ta2) {
		return false, "length1"
	}

	for i := 0; i < len(ta1); i++ {
		if ta1[i].EngineID != ta2[i].EngineID {
			return false, "engineids"
		}
		if ta1[i].ProcessedStats.ProcessedCPUMilliseconds != ta2[i].ProcessedStats.ProcessedCPUMilliseconds {
			return false, "processedStats.CPU"
		}
		if len(ta1[i].TaskOutput) != len(ta2[i].TaskOutput) {
			return false, "lenTaskOutputs"
		}
		for k, v := range ta1[i].TaskOutput {
			if v != ta2[i].TaskOutput[k] {
				return false, fmt.Sprintf("taskOutputKey=%s, ta1=%v, ta2=%v", k, ta1[i].TaskOutput[k], ta2[i].TaskOutput[k])
			}
		}
		if len(ta1[i].Inputs) != len(ta2[i].Inputs) {
			return false, "length of Inputs"
		}
		for j := 0; j < len(ta1[i].Inputs); j++ {
			v1 := ta1[i].Inputs[j]
			v2 := ta2[i].Inputs[j]
			if v1 != v2 {
				return false, fmt.Sprintf("Inputs at %d, v1=%v, v2=%v", j, v1, v2)
			}
		}
		if len(ta1[i].CreateJobsAction) != len(ta2[i].CreateJobsAction) {
			return false, "length of CreateJobActions"
		}
		for j := 0; j < len(ta1[i].CreateJobsAction); j++ {
			v1 := ta1[i].CreateJobsAction[j]
			v2 := ta2[i].CreateJobsAction[j]
			if v1.CreateTDODetail.Description != v2.CreateTDODetail.Description {
				return false, "createTDODetail-Description failed"
			}
			if v1.CreateTDODetail.StartDateTime.Unix() != v2.CreateTDODetail.StartDateTime.Unix() {
				return false, "createTDODetail-StartDateTime failed"
			}
			if v1.CreateTDODetail.StopDateTime.Unix() != v2.CreateTDODetail.StopDateTime.Unix() {
				return false, "createTDODetail-StartDateTime failed"
			}
		}
	}
	return true, ""
}
