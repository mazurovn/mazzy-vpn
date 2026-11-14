# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright © 2026 Nik m (@mazurovn). All rights reserved.
#
# Nim core of the nested HTTPS hop. It owns MCP. The inner TLS bytes are
# copied and never decrypted. The egress is never claimed clean.

import std/[asyncdispatch, asyncnet, base64, json, nativesockets, net, openssl, os,
            osproc, posix, sets, streams, strutils, times]

const
  pageBody = "<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>ok</title></head><body><p>ok</p></body></html>"
  prefaceLimit = 8192
  prefaceMs = 10_000

type
  Config = object
    bindAddr: string
    bindPort: int
    cert, key, credential: string
    idle, maxn, doctorEvery: int
    allows: seq[string]
    mcpSocket, statusSocket: string
  Session = ref object
    id: int
    phase: string
  Flow = ref object
    up, down: int
  Buf = ref object
    s: string

var
  sessions: seq[Session]
  nextId = 0
  relays = 0
  protectBin = ""
  stopping = false
  hopListener, mcpListener, statusListener: AsyncSocket

proc onStop(fd: AsyncFD): bool {.gcsafe.} =
  stopping = true
  if not hopListener.isNil:
    hopListener.close()
  if not mcpListener.isNil:
    mcpListener.close()
  if not statusListener.isNil:
    statusListener.close()
  true

proc die(code: int, msg: string) =
  stderr.writeLine(msg)
  quit(code)

proc ctEq(a, b: string): bool =
  var diff = a.len xor b.len
  let n = max(a.len, b.len)
  for i in 0 ..< n:
    let ca = if i < a.len: ord(a[i]) else: 0
    let cb = if i < b.len: ord(b[i]) else: 0
    diff = diff or (ca xor cb)
  diff == 0 and a.len == b.len

proc absPath(path: string): string =
  # absolutePath does not require the file to exist. expandFilename does,
  # and a missing config must be a refusal rather than an OS exception.
  absolutePath(path).normalizedPath

proc isUnder(path, root: string): bool =
  if root.len == 0:
    return false
  let p = absPath(path)
  let r = absPath(root)
  p == r or p.startsWith(r & "/")

proc lstatOf(path: string, st: var Stat): bool =
  lstat(path.cstring, st) == 0

proc symlinkInside(path: string): bool =
  var cur = ""
  for part in absPath(path).split('/'):
    if part.len == 0:
      continue
    cur = cur & "/" & part
    var st: Stat
    if not lstatOf(cur, st):
      return false
    if S_ISLNK(st.st_mode):
      return true
  false

proc fileTrusted(path: string, wantToeac: bool): bool =
  if path.contains(".."):
    return false
  let testRoot = getEnv("MAZZY_HOP_TEST_ROOT")
  let parent = path.parentDir
  let inEtc = absPath(parent) == "/etc/mazzy-vpn"
  let inTest = testRoot.len > 0 and isUnder(path, testRoot)
  if not inEtc and not inTest:
    return false
  if wantToeac and not path.endsWith(".toeac"):
    return false
  if symlinkInside(path):
    return false
  var st: Stat
  if not lstatOf(path, st):
    return false
  if S_ISLNK(st.st_mode) or not S_ISREG(st.st_mode):
    return false
  if (uint32(st.st_mode) and 0o777'u32) != 0o600'u32:
    return false
  var parentSt: Stat
  if not lstatOf(parent, parentSt) or S_ISLNK(parentSt.st_mode):
    return false
  if inEtc:
    return st.st_uid == Uid(0)
  st.st_uid == geteuid()

proc socketAllowed(path: string): bool =
  if path.len == 0 or path.contains("..") or not path.startsWith("/"):
    return false
  let testRoot = getEnv("MAZZY_HOP_TEST_ROOT")
  if symlinkInside(path):
    return false
  if testRoot.len > 0 and isUnder(path, testRoot):
    return true
  isUnder(path, "/run/mazzy-vpn") or isUnder(path, "/etc/mazzy-vpn")

proc runCapture(bin: string, args: seq[string], input: string): tuple[output: string, code: int] =
  let p = startProcess(bin, args = args, options = {poStdErrToStdOut})
  if input.len > 0:
    p.inputStream.write(input)
  p.inputStream.close()
  result.output = p.outputStream.readAll()
  result.code = p.waitForExit()
  p.close()

proc placeholder(s: string): bool =
  "REPLACE_" in s

proc parseFlat(text: string): Config =
  var
    sawOk = false
    listenOn, cert, keyPath, credential, mcpSocket, statusSocket: string
    idle, maxn, doctorEvery: int
    allows: seq[string]
    saw = initHashSet[string]()
    sawCore, sawBackend, sawFront, sawRelay, sawProtect, sawDoctor: bool
  for line in text.splitLines:
    if line.len == 0:
      continue
    let sp = line.find(' ')
    let key = if sp < 0: line else: line[0 ..< sp]
    let val = if sp < 0: "" else: line[sp + 1 .. ^1]
    if key notin ["allow", "ok", "artifact", "logic"] and saw.contains(key):
      die(78, "core: refused")
    if key notin ["ok", "artifact", "logic"]:
      saw.incl(key)
    case key
    of "ok":
      sawOk = true
    of "transport":
      if val != "nested-https":
        die(78, "core: refused")
    of "bind":
      listenOn = val
    of "cert":
      cert = val
    of "key":
      keyPath = val
    of "idle":
      idle = parseInt(val)
    of "max":
      maxn = parseInt(val)
    of "doctor":
      doctorEvery = parseInt(val)
    of "claim-clean":
      if val != "no":
        die(78, "core: refused")
    of "require-auth":
      if val != "yes":
        die(78, "core: refused")
    of "credential":
      credential = val
    of "allow":
      allows.add(val)
    of "mcp-socket":
      mcpSocket = val
    of "status-socket":
      statusSocket = val
    of "admin-bind", "admin-credential", "a2a-bind", "admin-url":
      discard
    of "artifact":
      if val == "mazzy-core mcp" and not sawCore:
        sawCore = true
      elif val == "mazzy-backend admin" and not sawBackend:
        sawBackend = true
      elif val == "mazzy-front a2a" and not sawFront:
        sawFront = true
      else:
        die(78, "core: refused")
    of "logic":
      if val == "relay copy" and not sawRelay:
        sawRelay = true
      elif val == "protect mojo" and not sawProtect:
        sawProtect = true
      elif val == "doctor nim" and not sawDoctor:
        sawDoctor = true
      else:
        die(78, "core: refused")
    else:
      die(78, "core: refused")
  if not sawOk or listenOn.len == 0 or cert.len == 0 or keyPath.len == 0 or
      credential.len < 16 or allows.len == 0 or idle < 1 or maxn < 1 or
      doctorEvery < 1 or mcpSocket.len == 0 or statusSocket.len == 0 or
      not sawCore or not sawBackend or not sawFront or not sawRelay or
      not sawProtect or not sawDoctor:
    die(78, "core: refused")
  for part in [listenOn, cert, keyPath, credential, mcpSocket, statusSocket]:
    if placeholder(part):
      die(78, "core: placeholder")
  for item in allows:
    if placeholder(item):
      die(78, "core: placeholder")
  let parts = listenOn.rsplit(':', 1)
  if parts.len != 2:
    die(78, "core: refused")
  var host = parts[0]
  if host.len >= 2 and host[0] == '[' and host[^1] == ']':
    host = host[1 .. ^2]
  result.bindAddr = host
  result.bindPort = parseInt(parts[1])
  result.cert = cert
  result.key = keyPath
  result.credential = credential
  result.idle = idle
  result.maxn = maxn
  result.doctorEvery = doctorEvery
  result.allows = allows
  result.mcpSocket = mcpSocket
  result.statusSocket = statusSocket

proc loadConfig(path, toeac: string): Config =
  if not fileTrusted(path, true):
    die(78, "core: refused")
  let ran = runCapture(toeac, @[path], "")
  if ran.code != 0:
    die(78, "core: toeac refused")
  parseFlat(ran.output)

proc splitHost(target: string): tuple[host: string, port: int] =
  let parts = target.rsplit(':', 1)
  if parts.len != 2:
    raise newException(ValueError, "target")
  result.host = parts[0]
  if result.host.len >= 2 and result.host[0] == '[' and result.host[^1] == ']':
    result.host = result.host[1 .. ^2]
  result.port = parseInt(parts[1])

proc allowed(cfg: Config, target: string): bool =
  for item in cfg.allows:
    if item == target:
      return true
  false

proc askProtect(authOk, allowOk, openRelays, limit: int): bool =
  if protectBin.len == 0:
    return false
  let line = "decide " & $authOk & " " & $allowOk & " 1 " & $openRelays & " " & $limit & "\n"
  let ran = runCapture(protectBin, @[], line)
  ran.code == 0 and ran.output.strip == "open"

proc traceOf(status: string, latency, up, down: int): string =
  let line = "trace " & status & " " & $latency & " " & $up & " " & $down & "\n"
  let ran = runCapture(protectBin, @[], line)
  if ran.code == 0 and "egress_clean=false" in ran.output and "egress_clean=true" notin ran.output:
    return ran.output.strip
  "status=refused latency_ms=0 bytes_up=0 bytes_down=0 egress_clean=false"

proc doctorReport(): string =
  "config trusted\nprotection on\nsessions " & $sessions.len &
    "\ntransport nested-https\negress_clean false\ntunnel independent"

proc statusJson(): string =
  $(%*{
    "egress_clean": false,
    "config": "trusted",
    "protection": "on",
    "sessions": sessions.len,
    "transport": "nested-https",
    "tunnel": "independent"
  })

proc toolText(): JsonNode =
  %*{
    "content": [{"type": "text", "text": doctorReport()}],
    "isError": false
  }

proc mcpResult(id, payload: JsonNode): string =
  $(%*{"jsonrpc": "2.0", "id": id, "result": payload})

proc mcpError(id: JsonNode, code: int, msg: string): string =
  $(%*{"jsonrpc": "2.0", "id": id, "error": {"code": code, "message": msg}})

proc handleMcp(body: string): string =
  var req: JsonNode
  try:
    req = parseJson(body)
  except:
    return mcpError(newJNull(), -32700, "refused")
  if req.kind != JObject or not req.hasKey("method"):
    let id = if req.kind == JObject and req.hasKey("id"): req["id"] else: newJNull()
    return mcpError(id, -32600, "refused")
  let rpc = req["method"].getStr
  let hasId = req.hasKey("id") and req["id"].kind != JNull
  let id = if req.hasKey("id"): req["id"] else: newJNull()
  if rpc.startsWith("notifications/"):
    return ""
  case rpc
  of "initialize":
    return mcpResult(id, %*{
      "protocolVersion": "2025-06-18",
      "capabilities": {"tools": {"listChanged": false}},
      "serverInfo": {"name": "mazzy-core", "version": "0.1.0"}
    })
  of "ping":
    return mcpResult(id, newJObject())
  of "tools/list":
    return mcpResult(id, %*{
      "tools": [
        {"name": "doctor", "description": "Health report. Does not claim the egress is clean."},
        {"name": "sessions", "description": "Open session count."},
        {"name": "protection", "description": "Whether protection is on."}
      ]
    })
  of "tools/call":
    var name = ""
    if req.hasKey("params") and req["params"].kind == JObject and req["params"].hasKey("name"):
      name = req["params"]["name"].getStr
    case name
    of "doctor", "sessions", "protection":
      return mcpResult(id, toolText())
    else:
      return mcpError(id, -32602, "refused")
  else:
    if not hasId:
      return ""
    return mcpError(id, -32601, "refused")

proc frame(body, raw: string): string =
  if body.len == 0:
    return ""
  if raw.startsWith("{") or raw.strip.startsWith("{"):
    return body & "\n"
  "Content-Length: " & $body.len & "\r\n\r\n" & body

proc readSome(sock: AsyncSocket, buf: Buf, limit, timeoutMs: int): Future[bool] {.async.} =
  if buf.s.len >= limit:
    return false
  let fut = sock.recv(1)
  if not await withTimeout(fut, timeoutMs):
    return false
  if fut.failed:
    return false
  let chunk = fut.read
  if chunk.len == 0:
    return false
  buf.s.add(chunk)
  return true

proc readMsg(sock: AsyncSocket): Future[tuple[raw, body: string]] {.async.} =
  let buf = Buf()
  while buf.s.len < 8:
    if not await readSome(sock, buf, prefaceLimit, prefaceMs):
      raise newException(IOError, "closed")
  let lead = buf.s.strip(leading = true, trailing = false)
  if lead.startsWith("{"):
    while not buf.s.endsWith("\n") and buf.s.len < prefaceLimit:
      if not await readSome(sock, buf, prefaceLimit, prefaceMs):
        break
    return (buf.s, buf.s.strip)
  while not buf.s.endsWith("\r\n\r\n") and buf.s.len < prefaceLimit:
    if not await readSome(sock, buf, prefaceLimit, prefaceMs):
      raise newException(IOError, "closed")
  var length = -1
  for line in buf.s.split("\r\n"):
    let low = line.toLowerAscii
    if low.startsWith("content-length:"):
      length = parseInt(line[line.find(':') + 1 .. ^1].strip)
  if length < 0 or length > 65536:
    raise newException(IOError, "length")
  var body = ""
  while body.len < length:
    let fut = sock.recv(length - body.len)
    let arrived = await withTimeout(fut, prefaceMs)
    if not arrived or fut.failed:
      raise newException(IOError, "body")
    let chunk = fut.read
    if chunk.len == 0:
      raise newException(IOError, "body")
    body.add(chunk)
  return (buf.s, body)

proc serveMcp(path: string) {.async.} =
  if not socketAllowed(path):
    die(78, "core: refused")
  var st: Stat
  if lstatOf(path, st):
    if S_ISLNK(st.st_mode) or not S_ISSOCK(st.st_mode):
      die(78, "core: refused")
    discard unlink(path.cstring)
  var server = newAsyncSocket(Domain.AF_UNIX, SockType.SOCK_STREAM, Protocol.IPPROTO_IP)
  server.bindUnix(path)
  discard chmod(path.cstring, 0o600)
  server.listen()
  mcpListener = server
  try:
    while not stopping:
      let client = await server.accept()
      asyncCheck (proc (c: AsyncSocket) {.async.} =
        try:
          while true:
            let msg = await readMsg(c)
            let body = handleMcp(msg.body)
            let outp = frame(body, msg.raw)
            if outp.len > 0:
              await c.send(outp)
        except:
          discard
        c.close()
      )(client)
  except:
    discard

proc serveStatus(path: string) {.async.} =
  if not socketAllowed(path):
    die(78, "core: refused")
  var st: Stat
  if lstatOf(path, st):
    if S_ISLNK(st.st_mode) or not S_ISSOCK(st.st_mode):
      die(78, "core: refused")
    discard unlink(path.cstring)
  var server = newAsyncSocket(Domain.AF_UNIX, SockType.SOCK_STREAM, Protocol.IPPROTO_IP)
  server.bindUnix(path)
  discard chmod(path.cstring, 0o600)
  server.listen()
  statusListener = server
  try:
    while not stopping:
      let client = await server.accept()
      asyncCheck (proc (c: AsyncSocket) {.async.} =
        try:
          let buf = Buf()
          while not buf.s.endsWith("\n") and buf.s.len < 64:
            if not await readSome(c, buf, 64, 2000):
              break
          let cmd = buf.s.strip
          if cmd in ["doctor", "sessions", "protection", "health"]:
            await c.send(statusJson() & "\n")
        except:
          discard
        c.close()
      )(client)
  except:
    discard

proc httpPage(verb: string): string =
  let head = "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: " &
    $pageBody.len & "\r\nConnection: close\r\n\r\n"
  if verb == "HEAD":
    head
  else:
    head & pageBody

proc passwordOf(preface: string): string =
  for line in preface.split("\r\n"):
    let low = line.toLowerAscii
    if low.startsWith("proxy-authorization:"):
      let value = line[line.find(':') + 1 .. ^1].strip
      if not value.toLowerAscii.startsWith("basic "):
        return ""
      try:
        let raw = decode(value[6 .. ^1].strip)
        let sp = raw.find(':')
        if sp <= 0:
          return ""
        return raw[sp + 1 .. ^1]
      except:
        return ""
  ""

proc requestParts(preface: string): tuple[verb, target: string] =
  let line = preface.split("\r\n")[0]
  let parts = line.split(' ')
  if parts.len >= 2:
    result.verb = parts[0].toUpperAscii
    result.target = parts[1]

proc pump(src, dst: AsyncSocket, idleMs: int, flow: Flow, outbound, oneByte: bool) {.async.} =
  # A buffered SSL socket fills the whole requested size before returning.
  # Ask for one byte, then drain what is already buffered, so a short TLS
  # record is copied instead of waiting for the window to fill.
  try:
    while not stopping:
      let size = if oneByte: 1 else: 16384
      let fut = src.recv(size)
      if not await withTimeout(fut, idleMs):
        break
      if fut.failed:
        break
      var chunk = fut.read
      if chunk.len == 0:
        break
      while chunk.len < 16384 and src.hasDataBuffered:
        let more = await src.recv(1)
        if more.len == 0:
          break
        chunk.add(more)
      if outbound:
        flow.up += chunk.len
      else:
        flow.down += chunk.len
      await dst.send(chunk)
  except:
    discard

proc splice(client, upstream: AsyncSocket, idleMs: int): Future[Flow] {.async.} =
  let flow = Flow()
  let a = pump(client, upstream, idleMs, flow, true, true)
  let b = pump(upstream, client, idleMs, flow, false, false)
  asyncCheck a
  asyncCheck b
  while not a.finished and not b.finished:
    await sleepAsync(20)
  client.close()
  upstream.close()
  return flow

proc forget(id: int) =
  var i = 0
  while i < sessions.len:
    if sessions[i].id == id:
      sessions.delete(i)
    else:
      inc i

proc handleClient(client: AsyncSocket, ctx: SslContext, cfg: Config) {.async.} =
  inc nextId
  let id = nextId
  let started = epochTime()
  let sess = Session(id: id, phase: "accepted")
  sessions.add(sess)
  var status = "hidden"
  var flow = Flow()
  try:
    if sessions.len > cfg.maxn * 4:
      status = "refused"
      client.close()
      return
    sess.phase = "handshake"
    wrapConnectedSocket(ctx, client, handshakeAsServer)
    sess.phase = "preface"
    let preface = Buf()
    let start = epochTime()
    while not preface.s.endsWith("\r\n\r\n") and preface.s.len < prefaceLimit:
      if int((epochTime() - start) * 1000) > prefaceMs:
        break
      if not await readSome(client, preface, prefaceLimit, prefaceMs):
        break
    let req = requestParts(preface.s)
    if req.verb != "CONNECT" or not preface.s.endsWith("\r\n\r\n"):
      await client.send(httpPage(req.verb))
      return
    let passwd = passwordOf(preface.s)
    let authOk = if ctEq(passwd, cfg.credential): 1 else: 0
    let allowOk = if allowed(cfg, req.target): 1 else: 0
    if not askProtect(authOk, allowOk, relays, cfg.maxn):
      status = "refused"
      await client.send(httpPage("GET"))
      return
    inc relays
    defer: dec relays
    sess.phase = "relaying"
    let hp = splitHost(req.target)
    # Unbuffered, so a short record is returned instead of waiting for 16 KiB.
    let upstream = await asyncnet.dial(hp.host, Port(hp.port), buffered = false)
    await client.send("HTTP/1.1 200 Connection Established\r\n\r\n")
    status = "established"
    flow = await splice(client, upstream, cfg.idle * 1000)
  except:
    if status == "established":
      status = "dial-failed"
    else:
      status = "refused"
  finally:
    forget(id)
    client.close()
    let ms = max(0, int((epochTime() - started) * 1000))
    stdout.writeLine(traceOf(status, ms, flow.up, flow.down))
    stdout.flushFile()

proc forceTls13(ctx: SslContext) =
  # OpenSSL exposes the minimum protocol as a control, not a separate symbol.
  if SSL_CTX_ctrl(ctx.context, 123.cint, 0x0304.clong, nil) != 1:
    die(78, "core: tls refused")

proc serveHop(cfg: Config, readyPath: string) {.async.} =
  if not fileTrusted(cfg.cert, false) or not fileTrusted(cfg.key, false):
    die(78, "core: refused")
  var ctx = newContext(certFile = cfg.cert, keyFile = cfg.key, verifyMode = CVerifyNone)
  forceTls13(ctx)
  var server = newAsyncSocket()
  server.setSockOpt(OptReuseAddr, true)
  server.bindAddr(Port(cfg.bindPort), cfg.bindAddr)
  server.listen()
  hopListener = server
  if readyPath.len > 0:
    writeFile(readyPath, "ready\n")
    discard chmod(readyPath.cstring, 0o600)
  try:
    while not stopping:
      let client = await server.accept()
      asyncCheck handleClient(client, ctx, cfg)
  except:
    discard

proc doctorLoop(every: int) {.async.} =
  while not stopping:
    await sleepAsync(every * 1000)
    if stopping:
      break
    stdout.writeLine(doctorReport())
    stdout.flushFile()

proc prepareSocket(path: string) =
  if not socketAllowed(path):
    die(78, "core: refused")
  if not dirExists(path.parentDir):
    die(78, "core: refused")

proc serve(cfg: Config, readyPath: string) {.async.} =
  prepareSocket(cfg.mcpSocket)
  prepareSocket(cfg.statusSocket)
  asyncCheck serveMcp(cfg.mcpSocket)
  asyncCheck serveStatus(cfg.statusSocket)
  asyncCheck doctorLoop(cfg.doctorEvery)
  await sleepAsync(30)
  await serveHop(cfg, readyPath)

proc trustMain(kind, path: string) =
  let toeac = kind == "trust-config"
  if fileTrusted(path, toeac):
    quit(0)
  die(78, "core: refused")

proc main() =
  if paramCount() >= 2 and paramStr(1) in ["trust-config", "trust-file"]:
    trustMain(paramStr(1), paramStr(2))
  if paramCount() < 1 or paramStr(1) != "serve":
    die(78, "core: refused")
  var
    configPath = ""
    toeac = ""
    protect = ""
    ready = ""
  var i = 2
  while i <= paramCount():
    let arg = paramStr(i)
    if i == paramCount():
      die(78, "core: refused")
    let val = paramStr(i + 1)
    case arg
    of "--config":
      configPath = val
    of "--toeac":
      toeac = val
    of "--protect":
      protect = val
    of "--ready":
      ready = val
    else:
      die(78, "core: refused")
    i += 2
  if configPath.len == 0 or toeac.len == 0 or protect.len == 0:
    die(78, "core: refused")
  protectBin = protect
  let cfg = loadConfig(configPath, toeac)
  addSignal(int(SIGTERM), onStop)
  addSignal(int(SIGINT), onStop)
  asyncCheck serve(cfg, ready)
  runForever()

when isMainModule:
  main()
