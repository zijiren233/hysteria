package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/apernet/hysteria/core/v2/server"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

var (
	_ server.Authenticator = (*V2boardApiProvider)(nil)
	_ server.TrafficLogger = (*V2boardApiProvider)(nil)
)

type trafficStatsEntry struct {
	Tx uint64 `json:"tx"`
	Rx uint64 `json:"rx"`
}

type V2boardApiProvider struct {
	client          *http.Client
	logger          *zap.Logger
	apiHost, apiKey string
	nodeID          uint
	usersMap        map[string]*user              // uuid -> user
	statsMap        map[string]*trafficStatsEntry // id -> stats
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
		statsMap: make(map[string]*trafficStatsEntry),
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

func (v *V2boardApiProvider) UpdateUsers(interval time.Duration) {
	v.logger.Info("用户列表自动更新服务已激活", zap.Duration("interval", interval))

	for {
		userList, err := v.getUserList(context.Background(), interval)
		if err != nil {
			v.logger.Error("获取用户列表失败", zap.Error(err))
			time.Sleep(time.Second * 15)
			continue
		}
		newUsersMap := make(map[string]*user, len(userList))
		for _, user := range userList {
			newUsersMap[user.UUID] = user
		}

		v.lock.Lock()
		v.usersMap = newUsersMap
		v.lock.Unlock()

		time.Sleep(interval)
	}
}

// 验证代码
func (v *V2boardApiProvider) Authenticate(addr net.Addr, auth string, tx uint64) (ok bool, uuid string) {
	// 获取判断连接用户是否在用户列表内
	v.lock.RLock()
	defer v.lock.RUnlock()

	if _, exists := v.usersMap[auth]; exists {
		return true, auth
	}
	v.logger.Debug("用户不存在", zap.String("auth", auth), zap.String("addr", addr.String()))
	return false, ""
}

func (v *V2boardApiProvider) LogTraffic(uuid string, tx uint64, rx uint64) bool {
	v.lock.Lock()
	defer v.lock.Unlock()

	user, ok := v.usersMap[uuid]
	if !ok {
		return false
	}

	entry, ok := v.statsMap[strconv.Itoa(user.ID)]
	if !ok {
		entry = &trafficStatsEntry{}
		v.statsMap[strconv.Itoa(user.ID)] = entry
	}
	entry.Tx += tx
	entry.Rx += rx

	return true
}

func (v *V2boardApiProvider) LogOnlineState(id string, online bool) {
}

type TrafficPushRequest struct {
	Data map[string][2]uint64
}

// 定时提交用户流量情况
func (v *V2boardApiProvider) PushTrafficToV2boardInterval(interval time.Duration) {
	v.logger.Info("用户流量情况监控已启动", zap.Duration("interval", interval))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := v.pushTrafficToV2board(
			fmt.Sprintf(
				"%s?token=%s&node_id=%d&node_type=hysteria",
				v.apiHost+"/api/v1/server/UniProxy/push",
				v.apiKey,
				v.nodeID,
			),
		); err != nil {
			v.logger.Error("提交用户流量情况失败", zap.Error(err))
		}
	}
}

// 向 v2board 提交用户流量使用情况
func (v *V2boardApiProvider) pushTrafficToV2board(url string) (err error) {
	v.lock.Lock()
	request := TrafficPushRequest{
		Data: make(map[string][2]uint64, len(v.statsMap)),
	}
	for id, stats := range v.statsMap {
		request.Data[id] = [2]uint64{stats.Tx, stats.Rx}
	}
	// 清空流量记录
	maps.Clear(v.statsMap)
	v.lock.Unlock()

	if len(request.Data) == 0 {
		return nil
	}

	defer func() {
		if err != nil {
			v.lock.Lock()
			defer v.lock.Unlock()
			for id, stats := range request.Data {
				entry, ok := v.statsMap[id]
				if !ok {
					entry = &trafficStatsEntry{}
					v.statsMap[id] = entry
				}
				entry.Tx += stats[0]
				entry.Rx += stats[1]
			}
		}
	}()

	// 将请求对象转换为JSON
	jsonData, err := json.Marshal(request.Data)
	if err != nil {
		return err
	}

	// 发起HTTP请求并提交数据
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP响应状态，处理错误等
	if resp.StatusCode != http.StatusOK {
		return errors.New("HTTP request failed with status code: " + resp.Status)
	}

	return nil
}
