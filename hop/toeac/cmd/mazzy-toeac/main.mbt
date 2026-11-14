async fn complain(code : Int32) -> Unit {
  let msg = if code == 1 { "toeac: refused\n" } else { "toeac: bad syntax\n" }
  @stdio.stderr.write(@utf8.encode(msg))
  @env.exit(code)
}

async fn main {
  let args = @env.args()
  if args.length() != 2 {
    complain(2)
    return
  }
  try {
    let src = @fs.read_text(args[1])
    match @toeac.render(src) {
      Ok(text) => println(text)
      Bad => complain(2)
      Refused => complain(1)
    }
  } catch {
    _ => complain(2)
  }
}
