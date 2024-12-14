import ReconnectingWebSocket from 'reconnecting-websocket';

class QWebSocket {
  constructor(url, options) {
    if (QWebSocket.instance) {
      return QWebSocket.instance;
    }

    this.url = url;
    this.options = options || {maxRetries: 10, reconnectInterval: 2000};
    this.ws = null; // WebSocket 实例
    this.callbacks = new Map(); // 保存 path 对应的回调函数
    this.requests = new Map(); // 保存请求的 Promise 和对应的 resolve/reject 函数
    this.requestId = 0; // 请求 ID 生成器

    if (url) {
      this.connect();

      // 缓存当前实例
      QWebSocket.instance = this;
    }
  }

  // 连接 WebSocket
  connect() {
    this.ws = new ReconnectingWebSocket(this.url, [], this.options);
    // WebSocket 事件绑定
    this.ws.onmessage = (event) => {
      this.handleMessage(event.data);
    };
  }

  // 处理收到的消息
  handleMessage(data) {
    try {
      const message = JSON.parse(data);
      const {path, requestId, payload} = message;

      // 如果是响应数据，根据 requestId 处理 Promise
      if (this.requests.has(requestId)) {
        const {resolve} = this.requests.get(requestId);
        resolve(payload);
        this.requests.delete(requestId);
        return;
      }

      // 如果有对应的回调函数，执行回调
      if (this.callbacks.has(path)) {
        this.callbacks.get(path)(payload);
      } else {
        console.warn(`No handler registered for path: ${path}`);
      }
    } catch (err) {
      console.error('Failed to handle message:', err);
    }
  }

  // 注册路径和回调函数
  on(path, callback) {
    if (typeof callback !== 'function') {
      throw new Error('Callback must be a function');
    }
    this.callbacks.set(path, callback);
  }

  remove(path) {
    if (this.callbacks.has(path)) {
      this.callbacks.delete(path);
    }
  }

  onopen(callback) {
    this.ws.onopen = callback
  }

  onclose(callback) {
    this.ws.onclose = callback
  }

  onerror(callback) {
    this.ws.onerror = callback
  }

  // 发送请求
  send(path, payload) {
    return new Promise((resolve, reject) => {
      if (this.ws.readyState !== WebSocket.OPEN) {
        return reject(new Error('WebSocket is not open'));
      }

      const requestId = ++this.requestId;
      this.requests.set(requestId, {resolve, reject});

      const message = JSON.stringify({path, requestId, payload});
      this.ws.send(message);
    });
  }

  // 关闭 WebSocket
  close() {
    this.ws.close();
  }

  get readyStatus() {
    return this.ws.readyState
  }
}

export default QWebSocket;
