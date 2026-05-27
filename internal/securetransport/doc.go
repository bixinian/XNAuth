// securetransport 包实现客户端接口层使用的请求/响应加密信封。
//
// 信封组合使用 X25519 密钥协商、HKDF-SHA256、AES-GCM 和 Ed25519 签名。
// 公钥可以分发给客户端，私钥只能保留在归属端。本包只负责传输层真实性和机密性，
// 业务授权仍由接口处理器在解开明文后执行。
package securetransport
