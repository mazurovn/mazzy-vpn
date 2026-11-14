# Protection decisions for the nested HTTPS hop.
# Numeric inputs only. This file does not see credentials, payloads, or names.
# It never reports the egress as clean.

def allowed_status(status: String) -> Bool:
    return (
        status == "established"
        or status == "refused"
        or status == "hidden"
        or status == "dial-failed"
        or status == "closed"
    )

def as_int(token: String) raises -> Int:
    return Int(token)

def decide(auth_ok: Int, allow_ok: Int, require_auth: Int, sessions: Int, limit: Int) -> String:
    if require_auth != 1:
        return "refuse"
    if auth_ok != 1 or allow_ok != 1:
        return "refuse"
    if sessions < 0 or limit < 1 or sessions >= limit:
        return "refuse"
    return "open"

def trace(status: String, latency_ms: Int, up: Int, down: Int) -> String:
    var safe = status
    if not allowed_status(safe):
        safe = "refused"
    if latency_ms < 0 or up < 0 or down < 0:
        safe = "refused"
        return "status=" + safe + " latency_ms=0 bytes_up=0 bytes_down=0 egress_clean=false"
    return (
        "status="
        + safe
        + " latency_ms="
        + String(latency_ms)
        + " bytes_up="
        + String(up)
        + " bytes_down="
        + String(down)
        + " egress_clean=false"
    )

def main() raises:
    var line = input().strip()
    var parts = line.split(" ")
    if len(parts) == 6 and parts[0] == "decide":
        try:
            print(
                decide(
                    as_int(String(parts[1])),
                    as_int(String(parts[2])),
                    as_int(String(parts[3])),
                    as_int(String(parts[4])),
                    as_int(String(parts[5])),
                )
            )
        except:
            print("refuse")
        return
    if len(parts) == 5 and parts[0] == "trace":
        try:
            print(trace(String(parts[1]), as_int(String(parts[2])), as_int(String(parts[3])), as_int(String(parts[4]))))
        except:
            print("status=refused latency_ms=0 bytes_up=0 bytes_down=0 egress_clean=false")
        return
    print("refuse")
