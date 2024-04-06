package auth

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/apernet/hysteria/core/server"
	"go.uber.org/zap"
)

var _ server.Authenticator = &V2boardApiProvider{}

type V2boardApiProvider struct {
	client          *http.Client
	logger          *zap.Logger
	apiHost, apiKey string
	nodeID          uint
	usersMap        map[string]*user
	lock            sync.RWMutex
}

func NewV2boardApiProvider(logger *zap.Logger, apiHost, apiKey string, nodeID uint) *V2boardApiProvider {
	return &V2boardApiProvider{
		client:   &http.Client{},
		logger:   logger,
		apiHost:  apiHost,
		apiKey:   apiKey,
		nodeID:   nodeID,
		usersMap: make(map[string]*user),
	}
}

type user struct {
	ID         int     `json:"id"`
	UUID       string  `json:"uuid"`
	SpeedLimit *uint32 `json:"speed_limit"`
}
type responseData struct {
	Users []*user `json:"users"`
}

func (v *V2boardApiProvider) getUserList(ctx context.Context, timeout time.Duration) ([]*user, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.apiHost+"/api/v1/server/UniProxy/user", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("token", v.apiKey)
	q.Add("node_id", strconv.Itoa(int(v.nodeID)))
	q.Add("node_type", "hysteria")
	req.URL.RawQuery = q.Encode()

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var responseData responseData
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	if err != nil {
		return nil, err
	}

	return responseData.Users, nil
}

func (v *V2boardApiProvider) UpdateUsers(interval time.Duration, trafficlogger server.TrafficLogger) {
	v.logger.Info("用户列表自动更新服务已激活")

	for {
		userList, err := v.getUserList(context.Background(), interval)
		if err != nil {
			v.logger.Error("获取用户列表失败", zap.Error(err))
			continue
		}
		newUsersMap := make(map[string]*user, len(userList))
		userIdList := make([]string, len(userList))
		for i, user := range userList {
			newUsersMap[user.UUID] = user
			userIdList[i] = strconv.Itoa(user.ID)
		}

		v.lock.Lock()
		v.usersMap = newUsersMap
		v.lock.Unlock()

		if trafficlogger != nil {
			trafficlogger.SetAllowedList(userIdList)
		}

		time.Sleep(interval)
	}
}

// 验证代码
func (v *V2boardApiProvider) Authenticate(addr net.Addr, auth string, tx uint64) (ok bool, id string) {
	// 获取判断连接用户是否在用户列表内
	v.lock.RLock()
	defer v.lock.RUnlock()

	if user, exists := v.usersMap[auth]; exists {
		return true, strconv.Itoa(user.ID)
	}
	v.logger.Warn("用户不存在", zap.String("auth", auth), zap.String("addr", addr.String()))
	return false, ""
}
