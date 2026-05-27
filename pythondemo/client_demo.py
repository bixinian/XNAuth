#!/usr/bin/env python3
"""
XNAuth Python 客户端示例。

运行后输入卡密和机器码，脚本会通过 /api/client/secure 发送加密请求，
完成授权验证后每 30 秒发送一次加密心跳。

注意：示例为了方便调试，会把客户端设备私钥明文保存到 device_key.json。
生产客户端应改用 Windows DPAPI、macOS Keychain、Linux Secret Service 等系统安全存储。
"""

from __future__ import annotations

import base64
import hashlib
import json
import os
import platform
import sys
import time
import urllib.error
import urllib.request
import uuid
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any
from urllib.parse import quote_plus


BASE_URL = os.getenv("XNAUTH_BASE_URL", "http://127.0.0.1:18080").rstrip("/")
APP_KEY = os.getenv("XNAUTH_APP_KEY", "").strip()
SERVER_KID = os.getenv("XNAUTH_SERVER_KID", "").strip()
SERVER_X25519_PUBLIC_KEY = os.getenv("XNAUTH_SERVER_X25519_PUBLIC_KEY", "").strip()
SERVER_ED25519_PUBLIC_KEY = os.getenv("XNAUTH_ED25519_PUBLIC_KEY", "").strip()
DEVICE_KEY_FILE = Path(os.getenv("XNAUTH_DEVICE_KEY_FILE", Path(__file__).with_name("device_key.json")))
CLIENT_VERSION = os.getenv("XNAUTH_CLIENT_VERSION", "1.0.0")
CLIENT_VERSION_CODE = int(os.getenv("XNAUTH_CLIENT_VERSION_CODE", "1"))
HEARTBEAT_INTERVAL_SECONDS = 30
REPORT_INTERVAL_SECONDS = 60


def main() -> int:
    print("XNAuth Python Secure Demo")
    print(f"服务地址: {BASE_URL}")

    try:
        crypto = load_crypto_modules()
    except Exception as exc:
        print(f"加载加密依赖失败: {exc}", file=sys.stderr)
        return 1

    app_key = APP_KEY or prompt_required("应用 App Key")
    try:
        secure_config = load_secure_config(app_key)
        device_key = load_or_create_device_key(crypto, DEVICE_KEY_FILE)
        client = SecureClient(crypto, secure_config, device_key)
    except Exception as exc:
        print(f"初始化安全客户端失败: {exc}", file=sys.stderr)
        return 1

    license_key = prompt_required("卡密")
    machine_code = prompt_required("机器码")
    device_name = prompt_optional("设备名", "Python Secure Demo")

    try:
        verify_data = verify(client, app_key, license_key, machine_code, device_name)
    except Exception as exc:
        print(f"授权验证失败: {exc}", file=sys.stderr)
        return 1

    session = verify_data.get("session", {})
    license_info = verify_data.get("license", {})
    session_token = str(session.get("session_token") or "")
    if not session_token:
        print("授权验证响应缺少 session_token", file=sys.stderr)
        return 1

    print(
        "授权验证成功: "
        f"license_id={license_info.get('license_id')} "
        f"device_id={license_info.get('device_id')} "
        f"max_devices={license_info.get('max_devices')} "
        f"max_online={license_info.get('max_online')}"
    )
    print(f"会话 token: {session_token}")
    print(f"设备公钥: {device_key.public_key_base64}")
    print(
        f"开始长运行: 心跳每 {HEARTBEAT_INTERVAL_SECONDS} 秒发送一次，"
        f"数据上报每 {REPORT_INTERVAL_SECONDS} 秒发送一次，按 Ctrl+C 停止。"
    )

    count = 0
    next_report_at = time.monotonic() + REPORT_INTERVAL_SECONDS
    try:
        while True:
            time.sleep(HEARTBEAT_INTERVAL_SECONDS)
            count += 1
            heartbeat(client, app_key, license_key, machine_code, session_token)
            print(f"[{now_text()}] 第 {count} 次加密心跳成功")

            if time.monotonic() >= next_report_at:
                report_data(client, app_key, license_key, machine_code, count)
                print(f"[{now_text()}] 数据上报成功")
                next_report_at = time.monotonic() + REPORT_INTERVAL_SECONDS
    except KeyboardInterrupt:
        print("\n已停止心跳。")
        return 0
    except Exception as exc:
        print(f"\n心跳失败，客户端退出: {exc}", file=sys.stderr)
        return 1


def verify(client: "SecureClient", app_key: str, license_key: str, machine_code: str, device_name: str) -> dict[str, Any]:
    payload = {
        "app_key": app_key,
        "license_key": license_key,
        "machine_code": machine_code,
        "device_name": device_name,
        "device_public_key": client.device_key.public_key_base64,
        "client_version": CLIENT_VERSION,
        "client_version_code": CLIENT_VERSION_CODE,
        "nonce": new_nonce("verify"),
    }
    return client.request("auth.verify", app_key, payload)


def heartbeat(
    client: "SecureClient",
    app_key: str,
    license_key: str,
    machine_code: str,
    session_token: str,
) -> dict[str, Any]:
    payload = {
        "app_key": app_key,
        "license_key": license_key,
        "session_token": session_token,
        "machine_code": machine_code,
        "client_version": CLIENT_VERSION,
        "client_version_code": CLIENT_VERSION_CODE,
        "nonce": new_nonce("heartbeat"),
    }
    return client.request("auth.heartbeat", app_key, payload)


def report_data(
    client: "SecureClient",
    app_key: str,
    license_key: str,
    machine_code: str,
    heartbeat_count: int,
) -> dict[str, Any]:
    payload = {
        "app_key": app_key,
        "license_key": license_key,
        "machine_code": machine_code,
        "event": "python_demo_report",
        "data": {
            "demo_client": "pythondemo",
            "wx_num": "99",
            "client_version": CLIENT_VERSION,
            "client_version_code": CLIENT_VERSION_CODE,
            "heartbeat_count": heartbeat_count,
            "hostname": platform.node(),
            "system": platform.system(),
            "release": platform.release(),
            "python_version": platform.python_version(),
            "reported_at": now_text(),
        },
        "nonce": new_nonce("collect"),
    }
    return client.request("collect.report", app_key, payload)


@dataclass
class CryptoModules:
    AESGCM: Any
    HKDF: Any
    hashes: Any
    serialization: Any
    InvalidSignature: type[Exception]
    ed25519: Any
    x25519: Any


@dataclass
class SecureConfig:
    kid: str
    server_x25519_public_key: str
    server_ed25519_public_key: str


@dataclass
class DeviceKey:
    private_key: Any
    public_key_base64: str


class SecureClient:
    def __init__(self, crypto: CryptoModules, config: SecureConfig, device_key: DeviceKey) -> None:
        self.crypto = crypto
        self.config = config
        self.device_key = device_key
        self.server_x25519_public_key = crypto.x25519.X25519PublicKey.from_public_bytes(
            b64decode_exact(config.server_x25519_public_key, 32, "服务端 X25519 公钥")
        )
        self.server_ed25519_public_key = crypto.ed25519.Ed25519PublicKey.from_public_bytes(
            b64decode_exact(config.server_ed25519_public_key, 32, "服务端 Ed25519 公钥")
        )

    def request(self, action: str, app_key: str, payload: dict[str, Any], device_id: int = 0) -> dict[str, Any]:
        nonce = str(payload.get("nonce") or new_nonce(action))
        payload["nonce"] = nonce
        timestamp = int(time.time())
        print_json(f"{action} 本地业务请求 JSON", payload)

        ephemeral_private = self.crypto.x25519.X25519PrivateKey.generate()
        ephemeral_public = ephemeral_private.public_key().public_bytes(
            encoding=self.crypto.serialization.Encoding.Raw,
            format=self.crypto.serialization.PublicFormat.Raw,
        )
        server_public = self.server_x25519_public_key.public_bytes(
            encoding=self.crypto.serialization.Encoding.Raw,
            format=self.crypto.serialization.PublicFormat.Raw,
        )
        shared_key = ephemeral_private.exchange(self.server_x25519_public_key)
        aes_key = derive_aes_key(self.crypto, shared_key, self.config.kid, ephemeral_public, server_public)

        body_nonce = os.urandom(12)
        envelope = {
            "v": 1,
            "kid": self.config.kid,
            "action": action,
            "app_key": app_key,
            "device_id": device_id,
            "timestamp": timestamp,
            "nonce": nonce,
            "body_nonce": b64encode(body_nonce),
            "client_ephemeral_public_key": b64encode(ephemeral_public),
        }

        plaintext = json.dumps(payload, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        # AES-GCM 的 AAD 绑定外层路由元数据，防止 action、nonce 或临时公钥被替换。
        ciphertext = self.crypto.AESGCM(aes_key).encrypt(body_nonce, plaintext, build_canonical_string(envelope).encode("utf-8"))
        envelope["ciphertext"] = b64encode(ciphertext)
        envelope["sign"] = b64encode(self.device_key.private_key.sign(self.request_signing_payload(envelope, ciphertext)))
        print_json(f"{action} 实际发送加密信封 JSON", envelope)

        response_envelope = post_json("/api/client/secure", envelope)
        print_json(f"{action} 服务端返回加密信封 JSON", response_envelope)
        response_body = self.open_response(aes_key, nonce, response_envelope)
        print_json(f"{action} 服务端返回解密业务 JSON", response_body)
        if response_body.get("code") != 0:
            raise RuntimeError(f"code={response_body.get('code')} message={response_body.get('message')}")
        data = response_body.get("data")
        return data if isinstance(data, dict) else {}

    def request_signing_payload(self, envelope: dict[str, Any], ciphertext: bytes) -> bytes:
        sign_data = dict(envelope)
        sign_data.pop("sign", None)
        sign_data.pop("ciphertext", None)
        sign_data["ciphertext_sha256"] = hashlib.sha256(ciphertext).hexdigest()
        return build_canonical_string(sign_data).encode("utf-8")

    def open_response(self, aes_key: bytes, request_nonce: str, envelope: dict[str, Any]) -> dict[str, Any]:
        required = ["v", "kid", "request_nonce", "nonce", "timestamp", "body_nonce", "ciphertext", "sign"]
        if any(key not in envelope for key in required):
            raise RuntimeError("服务端响应缺少安全信封字段")
        if envelope.get("kid") != self.config.kid or envelope.get("request_nonce") != request_nonce:
            raise RuntimeError("服务端响应信封不匹配")

        ciphertext = base64.b64decode(str(envelope["ciphertext"]))
        sign_payload = {
            "v": envelope["v"],
            "kid": envelope["kid"],
            "request_nonce": envelope["request_nonce"],
            "nonce": envelope["nonce"],
            "timestamp": envelope["timestamp"],
            "body_nonce": envelope["body_nonce"],
            "ciphertext_sha256": hashlib.sha256(ciphertext).hexdigest(),
        }
        try:
            self.server_ed25519_public_key.verify(
                base64.b64decode(str(envelope["sign"])),
                build_canonical_string(sign_payload).encode("utf-8"),
            )
        except self.crypto.InvalidSignature as exc:
            raise RuntimeError("服务端响应 Ed25519 签名校验失败") from exc

        response_aad = {
            "v": envelope["v"],
            "kid": envelope["kid"],
            "request_nonce": envelope["request_nonce"],
            "nonce": envelope["nonce"],
            "timestamp": envelope["timestamp"],
            "body_nonce": envelope["body_nonce"],
        }
        plaintext = self.crypto.AESGCM(aes_key).decrypt(
            base64.b64decode(str(envelope["body_nonce"])),
            ciphertext,
            build_canonical_string(response_aad).encode("utf-8"),
        )
        return json.loads(plaintext.decode("utf-8"))


def load_crypto_modules() -> CryptoModules:
    try:
        from cryptography.exceptions import InvalidSignature
        from cryptography.hazmat.primitives import hashes, serialization
        from cryptography.hazmat.primitives.asymmetric import ed25519, x25519
        from cryptography.hazmat.primitives.ciphers.aead import AESGCM
        from cryptography.hazmat.primitives.kdf.hkdf import HKDF
    except ModuleNotFoundError as exc:
        raise RuntimeError("缺少 cryptography 依赖，请先运行: pip install -r .\\pythondemo\\requirements.txt") from exc

    return CryptoModules(AESGCM, HKDF, hashes, serialization, InvalidSignature, ed25519, x25519)


def load_secure_config(app_key: str) -> SecureConfig:
    server_kid = SERVER_KID
    server_x25519_public_key = SERVER_X25519_PUBLIC_KEY
    server_ed25519_public_key = SERVER_ED25519_PUBLIC_KEY
    if not server_kid or not server_x25519_public_key or not server_ed25519_public_key:
        config = get_json(f"/api/client/secure/config?app_key={quote_plus(app_key)}").get("data", {})
        server_kid = server_kid or str(config.get("kid") or "")
        server_x25519_public_key = server_x25519_public_key or str(config.get("server_x25519_public_key") or "")
        server_ed25519_public_key = server_ed25519_public_key or str(config.get("server_ed25519_public_key") or "")
        print("已按 App Key 从服务端读取应用级安全传输公钥；生产客户端应内置或固定服务端公钥。")

    if not server_kid or not server_x25519_public_key or not server_ed25519_public_key:
        raise RuntimeError("安全传输配置不完整")
    return SecureConfig(server_kid, server_x25519_public_key, server_ed25519_public_key)


def load_or_create_device_key(crypto: CryptoModules, key_file: Path) -> DeviceKey:
    if key_file.exists():
        data = json.loads(key_file.read_text(encoding="utf-8"))
        private_key_raw = b64decode_exact(str(data.get("ed25519_private_key") or ""), 32, "客户端 Ed25519 私钥")
        private_key = crypto.ed25519.Ed25519PrivateKey.from_private_bytes(private_key_raw)
        print(f"已读取客户端设备密钥: {key_file}（Demo 明文保存；生产请使用系统安全存储）")
    else:
        private_key = crypto.ed25519.Ed25519PrivateKey.generate()
        key_file.parent.mkdir(parents=True, exist_ok=True)
        raw_private = private_key.private_bytes(
            encoding=crypto.serialization.Encoding.Raw,
            format=crypto.serialization.PrivateFormat.Raw,
            encryption_algorithm=crypto.serialization.NoEncryption(),
        )
        raw_public = private_key.public_key().public_bytes(
            encoding=crypto.serialization.Encoding.Raw,
            format=crypto.serialization.PublicFormat.Raw,
        )
        key_file.write_text(
            json.dumps(
                {
                    "ed25519_private_key": b64encode(raw_private),
                    "ed25519_public_key": b64encode(raw_public),
                },
                ensure_ascii=False,
                indent=2,
            ),
            encoding="utf-8",
        )
        print(f"已生成客户端设备密钥: {key_file}（Demo 明文保存；生产请使用系统安全存储）")

    raw_public = private_key.public_key().public_bytes(
        encoding=crypto.serialization.Encoding.Raw,
        format=crypto.serialization.PublicFormat.Raw,
    )
    return DeviceKey(private_key, b64encode(raw_public))


def derive_aes_key(crypto: CryptoModules, shared_key: bytes, kid: str, client_public: bytes, server_public: bytes) -> bytes:
    # Go 服务端使用同一套 HKDF 输入：salt = sha256(prefix + kid + client_pub + server_pub)。
    salt = hashlib.sha256(b"xnauth-secure-transport-salt:" + kid.encode("utf-8") + client_public + server_public).digest()
    return crypto.HKDF(
        algorithm=crypto.hashes.SHA256(),
        length=32,
        salt=salt,
        info=b"xnauth secure transport v1",
    ).derive(shared_key)


def post_json(path: str, payload: dict[str, Any]) -> dict[str, Any]:
    raw = json.dumps(payload, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
    request = urllib.request.Request(
        f"{BASE_URL}{path}",
        data=raw,
        method="POST",
        headers={"Content-Type": "application/json"},
    )
    return read_json_response(request)


def get_json(path: str) -> dict[str, Any]:
    request = urllib.request.Request(f"{BASE_URL}{path}", method="GET")
    return read_json_response(request)


def read_json_response(request: urllib.request.Request) -> dict[str, Any]:
    try:
        with urllib.request.urlopen(request, timeout=15) as response:
            body = response.read().decode("utf-8")
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(format_server_error(body, exc.code)) from exc
    except urllib.error.URLError as exc:
        raise RuntimeError(f"无法连接服务端: {exc.reason}") from exc

    try:
        return json.loads(body)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"服务端响应不是 JSON: {body}") from exc


def build_canonical_string(data: dict[str, Any]) -> str:
    return "&".join(f"{quote_plus(key)}={quote_plus(format_signature_value(data[key]))}" for key in sorted(data))


def print_json(title: str, payload: dict[str, Any]) -> None:
    print(f"\n--- {title} ---")
    print(json.dumps(payload, ensure_ascii=False, indent=2))


def format_signature_value(value: Any) -> str:
    if value is None:
        return ""
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, (int, float, str)):
        return str(value)
    if isinstance(value, bytes):
        return value.decode("utf-8")
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"))


def b64encode(raw: bytes) -> str:
    return base64.b64encode(raw).decode("ascii")


def b64decode_exact(value: str, length: int, label: str) -> bytes:
    try:
        raw = base64.b64decode(value)
    except ValueError as exc:
        raise RuntimeError(f"{label} 不是合法 Base64") from exc
    if len(raw) != length:
        raise RuntimeError(f"{label} 长度错误: {len(raw)}，期望 {length} 字节")
    return raw


def format_server_error(body: str, status_code: int) -> str:
    try:
        decoded = json.loads(body)
    except json.JSONDecodeError:
        return f"http_status={status_code} body={body}"
    return f"http_status={status_code} code={decoded.get('code')} message={decoded.get('message')}"


def prompt_required(label: str) -> str:
    while True:
        value = input(f"{label}: ").strip()
        if value:
            return value
        print(f"{label}不能为空")


def prompt_optional(label: str, default: str) -> str:
    value = input(f"{label} [{default}]: ").strip()
    return value or default


def new_nonce(prefix: str) -> str:
    return f"{prefix}-{uuid.uuid4().hex}"


def now_text() -> str:
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")


if __name__ == "__main__":
    raise SystemExit(main())
