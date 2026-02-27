# NaiveUI Subscription API Documentation

В этом документе описан формат данных, который ожидается клиентом NaiveUI от сервера (панели) при импорте подписок по URL, а также формат ручных лицензионных ключей.

## 1. Подписка по URL (Subscription URL)

Когда пользователь вводит URL подписки в клиенте (например, `https://panel.example.com/sub/ABC123XYZ`), клиент отправляет `GET` запрос по этому адресу со следующими заголовками:
```http
User-Agent: NaiveUI/1.0
X-HWID: <Hardware-ID-устройства>
```

Сервер в ответ должен вернуть валидный **JSON** со статусом `200 OK`. 
В случае любых других HTTP-статусов (400, 401, 403, 500 и т.д.) клиент выдаст ошибку с текстом тела ответа сервера.

### Формат JSON ответа (SubscriptionData)

```json
{
  "version": 1,
  "info": {
    "user_tag": "user@example.com",
    "expires_at": 1780000000,
    "traffic_limit_bytes": 107374182400,
    "traffic_used_bytes": 5368709120,
    "message": ""
  },
  "profiles": [
    {
      "name": "Server #1 (Frankfurt)",
      "server": "fr1.example.com",
      "port": 443,
      "username": "user_id_123",
      "password": "secret_password",
      "protocol": "https",
      "listen_protocol": "socks",
      "listen_port": 1080,
      "concurrency": 1,
      "extra_headers": ""
    }
  ]
}
```

### Описание полей

#### Корневой объект
- **`version`** *(integer, опционально, по умолчанию: 1)*: Версия формата ответа (зарезервировано на будущее).
- **`info`** *(object, опционально)*: Информация о пользователе и его лимитах. Если отсутствует, клиент будет считать подписку действующей бессрочно и безлимитной.
- **`profiles`** *(array of objects, обязательно)*: Список профилей (серверов), доступных пользователю. Может быть пустым масивом `[]`.

#### Объект `info` (SubscriptionInfo)
- **`user_tag`** *(string, опционально, по умолчанию: "")*: Имя, email или любой тег пользователя для отображения в интерфейсе клиента.
- **`expires_at`** *(integer, опционально, по умолчанию: 0)*: Unix timestamp времени окончания подписки в секундах. Значение `0` означает, что подписка бессрочная (не имеет срока давности).
- **`traffic_limit_bytes`** *(integer, опционально, по умолчанию: 0)*: Лимит трафика в байтах (на весь период или на месяц, в зависимости от логики вашей панели). Значение `0` означает безлимитный трафик.
- **`traffic_used_bytes`** *(integer, опционально, по умолчанию: 0)*: Уже израсходованный трафик в байтах. Используется для показа полосы прогресса в приложении.
- **`message`** *(string, опционально, по умолчанию: "")*: Сообщение от панели (например, предупреждение о скором исчерпании лимитов). Зарезервировано на будущее.

#### Объект внутри `profiles` массива (SubscriptionProfile)
Параметры, необходимые для подключения через NaiveProxy / sing-box.

- **`name`** *(string, обязательно)*: Название профиля, как оно будет отображаться в списке, например "Germany - FRA".
- **`server`** *(string, обязательно)*: Доменное имя или IP-адрес сервера NaiveProxy.
- **`port`** *(integer, обязательно)*: Порт сервера (как правило 443).
- **`username`** *(string, обязательно)*: Имя пользователя для HTTP Basic Auth к серверу.
- **`password`** *(string, обязательно)*: Пароль для HTTP Basic Auth к серверу.
- **`protocol`** *(string, обязательно)*: Протокол соединения NaiveProxy. Как правило, `"https"` или `"quic"`.
- **`listen_protocol`** *(string, опционально, по умолчанию: "socks")*: Протокол локального сервера. Обычно "socks", "http" или "mixed".
- **`listen_port`** *(integer, опционально, по умолчанию: 1080)*: Локальный порт для поднятия SOCKS5 прокси. Клиент может перезаписывать его при наличии конфликтов.
- **`concurrency`** *(integer, опционально, по умолчанию: 1)*: Параметр мультиплексирования (concurrency) / количество tcp-потоков в NaiveProxy.
- **`extra_headers`** *(string, опционально, по умолчанию: "")*: Дополнительные HTTP заголовки для включения в запросы (в формате "Key: Value\r\nKey: Value").

---

## 2. Ручные лицензионные ключи (Legacy Manual Keys)

Если вашей инфраструктуре требуется выдавать ключи вместо URL (например, для оффлайн использования или без разворачивания API панели), клиент поддерживает их ввод в то же самое поле.

### Формат ключа
Ключ представляет собой две Base64 URL Safe (без padding) строки, разделенные точкой (`.`):
`PayloadBase64` . `SignatureBase64`

- **PayloadBase64**: JSON строка, закодированная в base64url, содержащая структуру `LicenseData`.
- **SignatureBase64**: HMAC-SHA256 хеш от строки `PayloadBase64`, закодированный в base64url. Ключом для HMAC выступает секрет панели (`PANEL_SECRET`), жестко зашитый в `src-tauri/src/license.rs`.

### Структура `LicenseData` (Payload JSON)

Пример расшифрованного payload:
```json
{
  "hwid": "0AE1B2C3D4E5FA6B",
  "expires_at": 1780000000,
  "traffic_limit_bytes": 107374182400,
  "traffic_used_bytes": 5368709120,
  "user_tag": "user123",
  "panel_api_url": "https://panel.example.com/api/sync"
}
```

- **`hwid`** *(string)*: Уникальный ID устройства пользователя. Если оставить пустым `""`, ключ станет **универсальным** и привяжется к первому устройству, которое его импортирует.
- **`expires_at`** *(integer)*: Unix timestamp окончания времени действия (0 = бессрочно).
- **`traffic_limit_bytes`** *(integer)*: Лимит трафика в байтах (0 = безлимитно).
- **`traffic_used_bytes`** *(integer)*: Израсходованный трафик в байтах.
- **`user_tag`** *(string)*: Имя пользователя.
- **`panel_api_url`** *(string, опционально)*: URL для периодической синхронизации лимитов, можно оставить пустой строкой `""`.

### Генерация ключа (Пример на Node.js)

```javascript
const crypto = require('crypto');

const PANEL_SECRET = "NaivePanelSecret2024_ChangeMe_In_Production";

function generateKey(licenseData) {
    // 1. Кодируем JSON payload в Base64 URL-safe без паддинга (=)
    const payloadStr = JSON.stringify(licenseData);
    const payloadB64 = Buffer.from(payloadStr)
        .toString('base64')
        .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
    
    // 2. Считаем HMAC-SHA256
    const hmac = crypto.createHmac('sha256', PANEL_SECRET);
    hmac.update(payloadB64);
    const signatureBytes = hmac.digest();
    
    // 3. Кодируем подпись
    const sigB64 = signatureBytes
        .toString('base64')
        .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
        
    // 4. Формируем итоговый ключ
    return `${payloadB64}.${sigB64}`;
}

const key = generateKey({
    hwid: "", // Пустой HWID, чтобы клиент сам привязался при импорте
    expires_at: 0,
    traffic_limit_bytes: 0,
    traffic_used_bytes: 0,
    user_tag: "test_user",
    panel_api_url: ""
});
console.log("Ваш лицензионный ключ:\n" + key);
```
