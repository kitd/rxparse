export GOARCH=amd64
for os in windows linux darwin; do
  [ ${os} == "windows" ] && ext=".exe" || ext=""
  GOOS=${os} go build -o rxparse_${os}_${GOARCH}${ext} main.go
done