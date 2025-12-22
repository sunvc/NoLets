package controller

import (
	"sync"
	"time"

	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/push"
	"github.com/sunvc/apns2"
)

// MARK: - Push Task

var NotPushedDataList sync.Map

var oneDo sync.Once

func init() {
	oneDo.Do(CirclePush)
}

// CirclePush starts the periodic push task loop.
func CirclePush() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {

			NotPushedDataList.Range(func(key, value any) bool {
				data1, ok := value.(*common.NotPushedData)
				if !ok {
					NotPushedDataList.Delete(key) // Type assertion failed, delete
					return true
				}

				now := common.DateNow()

				// If not pushed for 24 hours, delete
				if now.Sub(data1.LastPushDate) > 24*time.Hour {
					NotPushedDataList.Delete(key)
					return true
				}

				// Push throttle strategy: wait Count × 10 minutes after each failure
				nextTry := data1.LastPushDate.Add(time.Duration(data1.Count) * 10 * time.Minute)
				if nextTry.After(now) {
					return true // Not time to push yet, skip
				}

				// Execute push
				if err := push.BatchPush(data1.Params, data1.PushType); err != nil {
					NotPushedDataList.Delete(key) // Push failed, delete
				}

				return true
			})
		}
	}()

}

// UpdateNotPushedData updates existing not-pushed data or adds a new record.
func UpdateNotPushedData(id string, params *common.ParamsResult, pushType apns2.EPushType) {
	if val, ok := NotPushedDataList.Load(id); ok {
		res := val.(*common.NotPushedData)
		res.LastPushDate = common.DateNow()
		res.Count++
		res.Params = params
		res.PushType = pushType
		NotPushedDataList.Store(id, common.Success) // Can be omitted, but kept for consistency
	} else {
		NotPushedDataList.Store(id, &common.NotPushedData{
			ID:           id,
			CreateDate:   common.DateNow(),
			LastPushDate: common.DateNow(),
			Count:        1,
			Params:       params,
			PushType:     pushType,
		})
	}
}
