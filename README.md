# ![Hysteria 2](logo.svg)

[![License][1]][2] [![Release][3]][4] [![Telegram][5]][6] [![Discussions][7]][8]

[1]: https://img.shields.io/badge/license-MIT-blue
[2]: LICENSE.md
[3]: https://img.shields.io/github/v/release/apernet/hysteria?style=flat-square
[4]: https://github.com/apernet/hysteria/releases
[5]: https://img.shields.io/badge/chat-Telegram-blue?style=flat-square
[6]: https://t.me/hysteria_github
[7]: https://img.shields.io/github/discussions/apernet/hysteria?style=flat-square
[8]: https://github.com/apernet/hysteria/discussions

<h2 style="text-align: center;">Hysteria is a powerful, lightning fast and censorship resistant proxy.</h2>

### [Get Started](https://v2.hysteria.network/)

### [中文文档](https://v2.hysteria.network/zh/)

### [Hysteria 1.x (legacy)](https://v1.hysteria.network/)

---

### 示例配置

```yaml
v2board:
  apiHost: https://面板地址
  apiKey: 面板节点密钥
  nodeID: 节点ID
tls:
  type: tls
  cert: /etc/hysteria/tls.crt
  key: /etc/hysteria/tls.key
auth:
  type: v2board
trafficStats:
  listen: 127.0.0.1:7653
acl: 
  inline: 
    - reject(10.0.0.0/8)
    - reject(172.16.0.0/12)
    - reject(192.168.0.0/16)
    - reject(127.0.0.0/8)
    - reject(fc00::/7)
```

---

<div class="feature-grid">
  <div>
    <h3>🛠️ Jack of all trades</h3>
    <p>Wide range of modes including SOCKS5, HTTP Proxy, TCP/UDP Forwarding, Linux TProxy, TUN - with more features being added constantly.</p>
  </div>

  <div>
    <h3>⚡ Blazing fast</h3>
    <p>Powered by a customized QUIC protocol, Hysteria is designed to deliver unparalleled performance over unreliable and lossy networks.</p>
  </div>

  <div>
    <h3>✊ Censorship resistant</h3>
    <p>The protocol masquerades as standard HTTP/3 traffic, making it very difficult for censors to detect and block without widespread collateral damage.</p>
  </div>
  
  <div>
    <h3>💻 Cross-platform</h3>
    <p>We have builds for every major platform and architecture. Deploy anywhere & use everywhere. Not to mention the long list of 3rd party apps.</p>
  </div>

  <div>
    <h3>🔗 Easy integration</h3>
    <p>With built-in support for custom authentication, traffic statistics & access control, Hysteria is easy to integrate into your infrastructure.</p>
  </div>
  
  <div>
    <h3>🤗 Cards on the table</h3>
    <p>We have well-documented specifications and code for developers to contribute and build their own apps. And a helpful community, too.</p>
  </div>
</div>

---

**If you find Hysteria useful, consider giving it a ⭐️!**

[![Star History Chart](https://api.star-history.com/svg?repos=apernet/hysteria&type=Date)](https://star-history.com/#apernet/hysteria&Date)
