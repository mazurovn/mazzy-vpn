# SDD — Control Plane contract: identity, trust, authorization (L4-0)

Роль: architect (opus5 route). Статус: SDD для реализации. Дата: 2026-08.

Расширяет `core/control` (сейчас — in-memory реестр + deny-by-default маршруты
«кто с кем») до полноценного контракта AI-native control plane.

## 1. Задача

Control plane соединяет агентов/харнессы/пользователей/приложения/устройства.
Сейчас есть только реестр участников и directional Allow/Revoke. Нет:
- **идентификации** (кто участник на самом деле);
- **доверия** (как устанавливается, что участнику можно верить);
- **авторизации** за пределами простого Allow (scopes, срок, отзыв).

## 2. Модель идентификации

### Participant identity
Каждый участник имеет:
- `ID` — стабильный, уникальный (уже есть);
- `Kind` — agent/harness/user/app/peer (уже есть);
- `PublicKey` — Ed25519 публичный ключ (НОВОЕ). ID рекомендуется выводить из
  ключа (`ID = base32(sha256(pubkey))[:16]`), чтобы ID был самозаверяющим.

### Самозаверяющий ID
`ID` привязан к ключу → нельзя зарегистрировать чужой ID без приватного ключа.
Регистрация подписывается приватным ключом участника (proof-of-possession).

## 3. Модель доверия

Три уровня, по возрастанию:

| Уровень | Как устанавливается | Что даёт |
|---|---|---|
| **untrusted** | просто зарегистрирован | ничего (deny-by-default) |
| **paired** | взаимный обмен ключами (out-of-band код/QR) | может быть целью Allow |
| **owned** | тот же владелец (общий root-ключ/устройство) | доверенные операции |

Доверие — НЕ транзитивно: A доверяет B, B доверяет C ≠ A доверяет C.

### Pairing (установление доверия)
Как в AdGuard/WireGuard: обмен публичными ключами по защищённому каналу
(QR-код, короткий код, файл). Каждая сторона добавляет ключ другой в свой
trust store. Подтверждается подписью challenge-response.

## 4. Модель авторизации

Расширяем `Route{from,to}` до `Grant`:

```
Grant {
  From, To   string      // participant IDs
  Scopes     []Scope     // что разрешено (не просто "reach")
  ExpiresAt  int64       // срок (0 = бессрочно)
  IssuedBy   string      // кто выдал (для аудита)
  Signature  []byte      // подпись выдающего
}
```

### Scopes
- `connect` — установить защищённый канал;
- `route:<provider>` — маршрутизировать к конкретному провайдеру/выходу;
- `control` — управлять (start/stop) целевым участником.

### Deny-by-default сохраняется
`CanReach(from,to,scope)` = есть НЕистёкший Grant с этим scope. Без гранта —
отказ (текущее поведение расширяется scope+expiry).

## 5. Отзыв (revocation)

- `Revoke(from,to)` — немедленно (уже есть);
- истечение `ExpiresAt` — автоматически;
- отзыв доверия (unpair) — каскадно отзывает все Grants к участнику.

## 6. Инварианты безопасности

1. **Self-authenticating ID**: нельзя выдать себя за другого без приватного ключа.
2. **Deny-by-default**: нет Grant → нет доступа.
3. **Нетранзитивность доверия**.
4. **Подписанные Grants**: получатель проверяет подпись выдающего.
5. **Срок + отзыв**: доступ не вечен; unpair рвёт всё.
6. **Аудит**: каждый Grant/Revoke логируется (mlog) с IssuedBy.

## 7. Что НЕ входит (осознанно)

- Транспорт pairing-канала (QR/код) — задача UI/L4-3.
- Реальный p2p data-plane между агентами — L4-3.
- CA/PKI-иерархия — избыточна; self-authenticating ключи достаточно.

## 8. План реализации (инкрементально)

| Шаг | Что | Тесты |
|---|---|---|
| L4-0a | `PublicKey` в Participant + self-auth ID (derive+verify) | derive, tamper |
| L4-0b | Trust store (untrusted/paired/owned) + pairing challenge | pair, non-transitive |
| L4-0c | `Grant` (scopes/expiry/signature) заменяет Route | expiry, scope, sig |
| L4-0d | Каскадный отзыв при unpair | revoke cascade |

## 9. Совместимость

Текущий `core/control` (Route/Allow) остаётся как упрощённый слой; Grant —
надстройка. Существующие тесты не ломаются (Allow = Grant со scope=connect,
без срока, локально доверенный).

## Вердикт

Контракт задаёт identity (Ed25519 self-auth), trust (3 уровня, нетранзитивно,
pairing), authorization (подписанные Grants со scope+expiry). Реализация
инкрементальная L4-0a..d, начиная со self-authenticating ID.
