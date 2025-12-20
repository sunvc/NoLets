package push

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/apns2"
	"github.com/sunvc/apns2/payload"
	"github.com/sunvc/apns2/token"
)

// Push message to APNs server
func Push(params *common.ParamsResult, pushType apns2.EPushType, token string) error {
	pl := payload.NewPayload().MutableContent()

	if pushType == apns2.PushTypeBackground {
		pl = pl.ContentAvailable()
	} else {

		pl = pl.AlertTitle(params.GetString(common.Title)).
			AlertSubtitle(params.GetString(common.Subtitle)).
			AlertBody(params.GetString(common.Body)).
			Sound(params.GetString(common.Sound)).
			TargetContentID(params.GetString(common.ID)).
			ThreadID(params.GetString(common.Group)).
			Category(params.GetString(common.Category))
	}

	// 添加自定义参数
	for pair := params.Params.Oldest(); pair != nil; pair = pair.Next() {
		if _, skip := common.SkipKeys[pair.Key]; skip {
			continue
		}
		pl.Custom(pair.Key, pair.Value)
	}

	CLI := <-CLIENTS // 从池中获取一个客户端
	CLIENTS <- CLI   // 将客户端放回池中

	// 创建并发送通知
	resp, err := CLI.Push(&apns2.Notification{
		DeviceToken: token,
		CollapseID:  params.GetString(common.ID),
		Topic:       common.LocalConfig.Apple.Topic,
		Payload:     pl,
		Expiration:  common.DateNow().Add(24 * time.Hour),
		PushType:    pushType,
	})

	// 错误处理
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("APNs push failed: %s", resp.Reason)
	}
	return nil

}

func BatchPush(params *common.ParamsResult, pushType apns2.EPushType) error {

	var (
		errors []error
		mu     sync.Mutex
		wg     sync.WaitGroup
	)

	for _, token := range params.Tokens {

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := Push(params, pushType, token); err != nil {
				log.Println(err.Error())
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	if len(errors) > 0 {
		return fmt.Errorf("APNs push failed: %v", errors)
	}

	return nil
}

func GetToken() (auth string, expirted int64) {
	CLI := <-CLIENTS // 从池中获取一个客户端
	CLIENTS <- CLI   // 将客户端放回池中
	return CLI.Token.GenerateIfExpired(), CLI.Token.IssuedAt + token.TokenTimeout
}
