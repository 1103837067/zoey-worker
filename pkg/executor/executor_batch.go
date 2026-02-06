package executor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	pb "github.com/zoeyai/zoeyworker/pkg/grpc/pb"
)

// ==================== 批量执行 ====================

// executeDebugCase 执行调试用例（顺序执行多个步骤）
func (e *Executor) executeDebugCase(taskID string, payload map[string]interface{}, startTime time.Time) {
	// 解析步骤列表
	stepsRaw, ok := payload["steps"].([]interface{})
	if !ok || len(stepsRaw) == 0 {
		taskErr := newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_PARAM_ERROR, "缺少 steps 参数或步骤列表为空")
		e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
		return
	}

	stopOnFail, _ := payload["stop_on_fail"].(bool)
	// 是否启用截图（默认启用，可通过 capture_screenshots: false 禁用）
	captureScreenshots := true
	if cs, ok := payload["capture_screenshots"].(bool); ok {
		captureScreenshots = cs
	}
	// 截图质量（JPEG 质量 1-100，默认 60 以减小传输量）
	screenshotQuality := 60
	if sq, ok := payload["screenshot_quality"].(float64); ok && sq > 0 && sq <= 100 {
		screenshotQuality = int(sq)
	}

	totalSteps := len(stepsRaw)

	log("INFO", fmt.Sprintf("[Task:%s] debug_case 开始，共 %d 个步骤, 截图=%v, 质量=%d", taskID, totalSteps, captureScreenshots, screenshotQuality))

	var completedSteps, passedSteps, failedSteps int32

	for i, stepRaw := range stepsRaw {
		stepMap, ok := stepRaw.(map[string]interface{})
		if !ok {
			log("WARN", fmt.Sprintf("[Task:%s] 步骤 %d 格式错误", taskID, i+1))
			continue
		}

		stepID, _ := stepMap["step_id"].(string)
		stepExecutionID, _ := stepMap["step_execution_id"].(string) // 步骤执行记录 ID（后端创建后传入）
		stepTaskType, _ := stepMap["task_type"].(string)
		stepParams, _ := stepMap["params"].(map[string]interface{})

		// 构建步骤级别的 taskID（用于前端区分每个步骤的结果）
		stepTaskID := fmt.Sprintf("step_%s_%d", stepID, time.Now().UnixMilli())

		log("INFO", fmt.Sprintf("[Task:%s] 执行步骤 %d/%d: %s (type=%s)", taskID, i+1, totalSteps, stepID, stepTaskType))

		// 发送步骤进度
		e.sendTaskProgress(taskID, int32(totalSteps), completedSteps, passedSteps, failedSteps, stepTaskType, "RUNNING")

		// 执行步骤（带前后截图）
		stepResult := e.executeStepWithScreenshots(stepExecutionID, stepID, stepTaskType, stepParams, captureScreenshots, screenshotQuality)

		completedSteps++

		if stepResult.Status != "SUCCESS" {
			failedSteps++
			log("ERROR", fmt.Sprintf("[Task:%s] 步骤 %s 执行失败: %s", taskID, stepID, stepResult.ErrorMessage))

			// 发送步骤失败结果（使用增强版）
			e.sendStepResultV2(stepTaskID, stepResult)

			if stopOnFail {
				log("INFO", fmt.Sprintf("[Task:%s] stop_on_fail=true，停止执行", taskID))
				// 发送整体任务失败结果
				e.sendTaskProgress(taskID, int32(totalSteps), completedSteps, passedSteps, failedSteps, stepTaskType, "FAILED")
				taskErr := newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_UNSPECIFIED, stepResult.ErrorMessage)
				e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
				return
			}
		} else {
			passedSteps++
			log("INFO", fmt.Sprintf("[Task:%s] 步骤 %s 执行成功", taskID, stepID))

			// 发送步骤成功结果（使用增强版）
			e.sendStepResultV2(stepTaskID, stepResult)
		}
	}

	// 所有步骤执行完成
	log("INFO", fmt.Sprintf("[Task:%s] debug_case 完成: passed=%d, failed=%d", taskID, passedSteps, failedSteps))

	// 发送最终进度和结果
	finalStatus := "SUCCESS"
	if failedSteps > 0 {
		finalStatus = "PARTIAL_FAILED"
	}
	e.sendTaskProgress(taskID, int32(totalSteps), completedSteps, passedSteps, failedSteps, "", finalStatus)

	// 发送整体任务结果
	resultJSON, _ := json.Marshal(map[string]interface{}{
		"total_steps":     totalSteps,
		"completed_steps": completedSteps,
		"passed_steps":    passedSteps,
		"failed_steps":    failedSteps,
	})

	if failedSteps > 0 {
		e.sendTaskResultWithError(taskID, newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_UNSPECIFIED, fmt.Sprintf("部分步骤失败: %d/%d", failedSteps, totalSteps)), nil, startTime)
	} else {
		e.sendTaskResultSuccess(taskID, string(resultJSON), nil, startTime)
	}
}

// executeExecutePlan 执行测试计划（顺序执行多个用例）
// payload 格式:
//
//	{
//	  "plan_execution_id": "xxx",
//	  "plan_id": "xxx",
//	  "cases": [
//	    {
//	      "case_execution_id": "xxx",
//	      "case_id": "xxx",
//	      "case_name": "用例名称",
//	      "steps": [...]  // 同 debug_case 格式
//	    }
//	  ],
//	  "stop_on_fail": true/false,
//	  "capture_screenshots": true/false,
//	  "screenshot_quality": 60
//	}
func (e *Executor) executeExecutePlan(taskID string, payload map[string]interface{}, startTime time.Time) {
	planExecutionID, _ := payload["plan_execution_id"].(string)
	planID, _ := payload["plan_id"].(string)

	// 解析用例列表
	casesRaw, ok := payload["cases"].([]interface{})
	if !ok || len(casesRaw) == 0 {
		taskErr := newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_PARAM_ERROR, "缺少 cases 参数或用例列表为空")
		e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
		return
	}

	stopOnFail, _ := payload["stop_on_fail"].(bool)
	captureScreenshots := true
	if cs, ok := payload["capture_screenshots"].(bool); ok {
		captureScreenshots = cs
	}
	screenshotQuality := 60
	if sq, ok := payload["screenshot_quality"].(float64); ok && sq > 0 && sq <= 100 {
		screenshotQuality = int(sq)
	}

	totalCases := len(casesRaw)
	log("INFO", fmt.Sprintf("[Task:%s] execute_plan 开始，计划=%s，共 %d 个用例", taskID, planID, totalCases))

	var completedCases, passedCases, failedCases int32

	for caseIdx, caseRaw := range casesRaw {
		caseMap, ok := caseRaw.(map[string]interface{})
		if !ok {
			log("WARN", fmt.Sprintf("[Task:%s] 用例 %d 格式错误", taskID, caseIdx+1))
			continue
		}

		caseExecutionID, _ := caseMap["case_execution_id"].(string)
		caseID, _ := caseMap["case_id"].(string)
		caseName, _ := caseMap["case_name"].(string)
		stepsRaw, _ := caseMap["steps"].([]interface{})

		if len(stepsRaw) == 0 {
			log("WARN", fmt.Sprintf("[Task:%s] 用例 %s 没有步骤，跳过", taskID, caseName))
			continue
		}

		log("INFO", fmt.Sprintf("[Task:%s] 执行用例 %d/%d: %s (id=%s)", taskID, caseIdx+1, totalCases, caseName, caseID))

		// 执行用例中的所有步骤
		caseResult := e.executeCaseSteps(taskID, caseExecutionID, caseID, stepsRaw, stopOnFail, captureScreenshots, screenshotQuality)

		completedCases++
		if caseResult.Success {
			passedCases++
			log("INFO", fmt.Sprintf("[Task:%s] 用例 %s 执行成功", taskID, caseName))
		} else {
			failedCases++
			log("ERROR", fmt.Sprintf("[Task:%s] 用例 %s 执行失败: %s", taskID, caseName, caseResult.ErrorMessage))

			if stopOnFail {
				log("INFO", fmt.Sprintf("[Task:%s] stop_on_fail=true，停止执行计划", taskID))
				break
			}
		}
	}

	// 所有用例执行完成
	log("INFO", fmt.Sprintf("[Task:%s] execute_plan 完成: passed=%d, failed=%d", taskID, passedCases, failedCases))

	// 发送整体结果
	resultJSON, _ := json.Marshal(map[string]interface{}{
		"plan_execution_id": planExecutionID,
		"plan_id":           planID,
		"total_cases":       totalCases,
		"completed_cases":   completedCases,
		"passed_cases":      passedCases,
		"failed_cases":      failedCases,
	})

	if failedCases > 0 {
		e.sendTaskResultWithError(taskID, newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_UNSPECIFIED, fmt.Sprintf("部分用例失败: %d/%d", failedCases, totalCases)), nil, startTime)
	} else {
		e.sendTaskResultSuccess(taskID, string(resultJSON), nil, startTime)
	}
}

// executeCaseSteps 执行用例中的所有步骤（内部方法，供 execute_plan 和 execute_case 使用）
func (e *Executor) executeCaseSteps(taskID, caseExecutionID, caseID string, stepsRaw []interface{}, stopOnFail, captureScreenshots bool, screenshotQuality int) *CaseExecutionResult {
	result := &CaseExecutionResult{
		Success:    true,
		TotalSteps: len(stepsRaw),
	}

	for i, stepRaw := range stepsRaw {
		stepMap, ok := stepRaw.(map[string]interface{})
		if !ok {
			log("WARN", fmt.Sprintf("[Task:%s] 步骤 %d 格式错误", taskID, i+1))
			continue
		}

		stepID, _ := stepMap["step_id"].(string)
		stepExecutionID, _ := stepMap["step_execution_id"].(string)
		stepTaskType, _ := stepMap["task_type"].(string)
		stepParams, _ := stepMap["params"].(map[string]interface{})

		// 构建步骤级别的 taskID
		stepTaskID := fmt.Sprintf("step_%s_%d", stepID, time.Now().UnixMilli())

		log("INFO", fmt.Sprintf("[Task:%s] 执行步骤 %d/%d: %s (type=%s)", taskID, i+1, len(stepsRaw), stepID, stepTaskType))

		// 发送步骤进度
		e.sendTaskProgress(taskID, int32(len(stepsRaw)), int32(i), int32(result.PassedSteps), int32(result.FailedSteps), stepTaskType, "RUNNING")

		// 执行步骤（带前后截图）
		stepResult := e.executeStepWithScreenshots(stepExecutionID, stepID, stepTaskType, stepParams, captureScreenshots, screenshotQuality)

		if stepResult.Status != "SUCCESS" {
			result.FailedSteps++
			taskErr := classifyError(fmt.Errorf("%s", stepResult.ErrorMessage))

			// 发送步骤失败结果
			e.sendStepResultV2(stepTaskID, stepResult)

			if stopOnFail {
				result.Success = false
				result.ErrorMessage = taskErr.Message
				return result
			}
		} else {
			result.PassedSteps++

			// 发送步骤成功结果
			e.sendStepResultV2(stepTaskID, stepResult)
		}
	}

	// 如果有失败的步骤，标记用例失败
	if result.FailedSteps > 0 {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("部分步骤失败: %d/%d", result.FailedSteps, result.TotalSteps)
	}

	return result
}

// executeExecuteCase 执行单个用例
// payload 格式同 debug_case，但会保存执行记录到数据库
func (e *Executor) executeExecuteCase(taskID string, payload map[string]interface{}, startTime time.Time) {
	caseExecutionID, _ := payload["case_execution_id"].(string)
	caseID, _ := payload["case_id"].(string)

	// 解析步骤列表
	stepsRaw, ok := payload["steps"].([]interface{})
	if !ok || len(stepsRaw) == 0 {
		taskErr := newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_PARAM_ERROR, "缺少 steps 参数或步骤列表为空")
		e.sendTaskResultWithError(taskID, taskErr, nil, startTime)
		return
	}

	stopOnFail := true // 默认遇到失败停止
	if sf, ok := payload["stop_on_fail"].(bool); ok {
		stopOnFail = sf
	}
	captureScreenshots := true
	if cs, ok := payload["capture_screenshots"].(bool); ok {
		captureScreenshots = cs
	}
	screenshotQuality := 60
	if sq, ok := payload["screenshot_quality"].(float64); ok && sq > 0 && sq <= 100 {
		screenshotQuality = int(sq)
	}

	log("INFO", fmt.Sprintf("[Task:%s] execute_case 开始，用例=%s，共 %d 个步骤", taskID, caseID, len(stepsRaw)))

	// 执行所有步骤
	result := e.executeCaseSteps(taskID, caseExecutionID, caseID, stepsRaw, stopOnFail, captureScreenshots, screenshotQuality)

	log("INFO", fmt.Sprintf("[Task:%s] execute_case 完成: passed=%d, failed=%d", taskID, result.PassedSteps, result.FailedSteps))

	// 发送结果
	resultJSON, _ := json.Marshal(map[string]interface{}{
		"case_execution_id": caseExecutionID,
		"case_id":           caseID,
		"total_steps":       result.TotalSteps,
		"passed_steps":      result.PassedSteps,
		"failed_steps":      result.FailedSteps,
	})

	if !result.Success {
		e.sendTaskResultWithError(taskID, newTaskError(pb.TaskStatus_TASK_STATUS_FAILED, pb.FailureReason_FAILURE_REASON_UNSPECIFIED, result.ErrorMessage), nil, startTime)
	} else {
		e.sendTaskResultSuccess(taskID, string(resultJSON), nil, startTime)
	}
}

// ==================== 步骤截图执行 ====================

// executeStepWithScreenshots 执行单个步骤并在前后截图
// 返回完整的 StepExecutionResult，供 executeDebugCase 和 executeCaseSteps 共用
func (e *Executor) executeStepWithScreenshots(
	stepExecutionID, stepID, stepTaskType string,
	stepParams map[string]interface{},
	captureScreenshots bool, screenshotQuality int,
) *StepExecutionResult {
	// 1. 执行前截图
	var screenshotBefore string
	if captureScreenshots {
		if sb, err := auto.CaptureScreenToBase64(screenshotQuality); err == nil {
			screenshotBefore = sb
		}
	}

	// 2. 执行步骤
	stepStartTime := time.Now()
	actionResult := e.executeSingleStepV2(stepTaskType, stepParams)
	durationMs := time.Since(stepStartTime).Milliseconds()

	// 3. 执行后截图
	var screenshotAfter string
	if captureScreenshots {
		if sa, err := auto.CaptureScreenToBase64(screenshotQuality); err == nil {
			screenshotAfter = sa
		}
	}

	// 4. 构建步骤执行结果
	stepResult := &StepExecutionResult{
		StepExecutionID:  stepExecutionID,
		StepID:           stepID,
		ActionType:       mapTaskTypeToActionType(stepTaskType),
		ScreenshotBefore: screenshotBefore,
		ScreenshotAfter:  screenshotAfter,
		TargetBounds:     actionResult.TargetBounds,
		ClickPosition:    actionResult.ClickPosition,
		InputText:        actionResult.InputText,
		DurationMs:       durationMs,
	}

	// 提取脚本执行输出（Python 等）
	if actionResult.Data != nil {
		if dataMap, ok := actionResult.Data.(map[string]interface{}); ok {
			if stdout, ok := dataMap["stdout"].(string); ok {
				stepResult.Stdout = stdout
			}
			if stderr, ok := dataMap["stderr"].(string); ok {
				stepResult.Stderr = stderr
			}
			if exitCode, ok := dataMap["exit_code"].(int); ok {
				stepResult.ExitCode = exitCode
			} else if exitCode, ok := dataMap["exit_code"].(float64); ok {
				stepResult.ExitCode = int(exitCode)
			}
		}
	}

	if !actionResult.Success {
		taskErr := classifyError(actionResult.Error)
		stepResult.Status = mapTaskStatusToString(taskErr.Status)
		stepResult.ErrorMessage = taskErr.Message
		stepResult.FailureReason = mapFailureReasonToString(taskErr.Reason)
	} else {
		stepResult.Status = "SUCCESS"
	}

	return stepResult
}

// ==================== 结果发送 ====================

// sendTaskProgress 发送任务进度
func (e *Executor) sendTaskProgress(taskID string, totalSteps, completedSteps, passedSteps, failedSteps int32, currentStepName, status string) {
	if e.client == nil {
		return
	}

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("progress_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		Payload: &pb.WorkerMessage_TaskProgress{
			TaskProgress: &pb.TaskProgress{
				TaskId:          taskID,
				TotalSteps:      totalSteps,
				CompletedSteps:  completedSteps,
				PassedSteps:     passedSteps,
				FailedSteps:     failedSteps,
				CurrentStepName: currentStepName,
				Status:          status,
			},
		},
	}

	e.client.SendTaskMessage(msg)
}

// sendStepResultV2 发送单个步骤的执行结果（增强版，包含完整的回放数据）
func (e *Executor) sendStepResultV2(taskID string, result *StepExecutionResult) {
	if e.client == nil {
		return
	}

	// 序列化完整的步骤执行结果
	resultJSON, _ := json.Marshal(result)

	// 确定任务状态和失败原因
	var status pb.TaskStatus
	var failureReason pb.FailureReason
	success := result.Status == "SUCCESS"

	switch result.Status {
	case "SUCCESS":
		status = pb.TaskStatus_TASK_STATUS_SUCCESS
		failureReason = pb.FailureReason_FAILURE_REASON_UNSPECIFIED
	case "FAILED":
		status = pb.TaskStatus_TASK_STATUS_FAILED
		switch result.FailureReason {
		case "NOT_FOUND":
			failureReason = pb.FailureReason_FAILURE_REASON_NOT_FOUND
		case "MULTIPLE_MATCHES":
			failureReason = pb.FailureReason_FAILURE_REASON_MULTIPLE_MATCHES
		case "ASSERTION_FAILED":
			failureReason = pb.FailureReason_FAILURE_REASON_ASSERTION_FAILED
		case "PARAM_ERROR":
			failureReason = pb.FailureReason_FAILURE_REASON_PARAM_ERROR
		case "SYSTEM_ERROR":
			failureReason = pb.FailureReason_FAILURE_REASON_SYSTEM_ERROR
		default:
			failureReason = pb.FailureReason_FAILURE_REASON_UNSPECIFIED
		}
	case "SKIPPED":
		status = pb.TaskStatus_TASK_STATUS_SKIPPED
		failureReason = pb.FailureReason_FAILURE_REASON_UNSPECIFIED
	default:
		status = pb.TaskStatus_TASK_STATUS_FAILED
		failureReason = pb.FailureReason_FAILURE_REASON_UNSPECIFIED
	}

	// 构建 MatchLocation（如果有目标边界信息）
	var matchLoc *pb.MatchLocation
	if result.TargetBounds != nil {
		matchLoc = &pb.MatchLocation{
			X:      int32(result.TargetBounds.X),
			Y:      int32(result.TargetBounds.Y),
			Width:  int32(result.TargetBounds.Width),
			Height: int32(result.TargetBounds.Height),
		}
	}

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("step_result_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		Payload: &pb.WorkerMessage_TaskResult{
			TaskResult: &pb.TaskResult{
				TaskId:        taskID,
				Success:       success,
				Status:        status,
				Message:       result.ErrorMessage,
				ResultJson:    string(resultJSON),
				DurationMs:    result.DurationMs,
				FailureReason: failureReason,
				MatchLocation: matchLoc,
			},
		},
	}

	e.client.SendTaskMessage(msg)
}

// sendTaskAck 发送任务确认
func (e *Executor) sendTaskAck(taskID string, accepted bool, message string) {
	if e.client == nil {
		return
	}

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("ack_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		Payload: &pb.WorkerMessage_TaskAck{
			TaskAck: &pb.TaskAck{
				TaskId:   taskID,
				Accepted: accepted,
				Message:  message,
			},
		},
	}

	e.client.SendTaskMessage(msg)
}

// sendTaskResultSuccess 发送成功结果
func (e *Executor) sendTaskResultSuccess(taskID string, resultJSON string, matchLoc *pb.MatchLocation, startTime time.Time) {
	if e.client == nil {
		return
	}

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("result_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		Payload: &pb.WorkerMessage_TaskResult{
			TaskResult: &pb.TaskResult{
				TaskId:        taskID,
				Success:       true,
				Status:        pb.TaskStatus_TASK_STATUS_SUCCESS,
				Message:       "",
				ResultJson:    resultJSON,
				DurationMs:    time.Since(startTime).Milliseconds(),
				FailureReason: pb.FailureReason_FAILURE_REASON_UNSPECIFIED,
				MatchLocation: matchLoc,
			},
		},
	}

	e.client.SendTaskMessage(msg)
}

// sendTaskResultWithError 发送失败结果
// 可选的 resultJSON 参数允许在失败时也附带执行数据（如 Python 的 stdout/stderr）
func (e *Executor) sendTaskResultWithError(taskID string, taskErr *TaskError, matchLoc *pb.MatchLocation, startTime time.Time, resultJSON ...string) {
	if e.client == nil {
		return
	}

	// 使用传入的 resultJSON 或默认空对象
	rj := "{}"
	if len(resultJSON) > 0 && resultJSON[0] != "" {
		rj = resultJSON[0]
	}

	msg := &pb.WorkerMessage{
		MessageId: fmt.Sprintf("result_%d", time.Now().UnixMilli()),
		Timestamp: time.Now().UnixMilli(),
		Payload: &pb.WorkerMessage_TaskResult{
			TaskResult: &pb.TaskResult{
				TaskId:        taskID,
				Success:       false,
				Status:        taskErr.Status,
				Message:       taskErr.Message,
				ResultJson:    rj,
				DurationMs:    time.Since(startTime).Milliseconds(),
				FailureReason: taskErr.Reason,
				MatchLocation: matchLoc,
			},
		},
	}

	e.client.SendTaskMessage(msg)
}
