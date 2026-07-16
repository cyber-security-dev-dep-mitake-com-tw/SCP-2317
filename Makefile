.PHONY: test run test-robot build-addin zip

test:
	go test ./...

run:
	go run ./cmd/api

test-robot:
	bash scripts/gitlab/run-robot.sh

build-addin:
	cd outlook-addin && npm install && npm run build

zip:
	cd .. && zip -qr scp2317-agent.zip SCP-2317 -x '*/node_modules/*' '*/target/*' '*/.build/*' '*/.git/*' '*/dist/*' '*/robot-output/*'
