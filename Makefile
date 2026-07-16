.PHONY: test run zip
test:
	go test ./...
run:
	go run ./cmd/api
zip:
	cd .. && zip -qr scp2317-agent.zip scp-2317-the-door-to-another-dimension -x '*/node_modules/*' '*/target/*' '*/.git/*'
