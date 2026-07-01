# c48x-airbridge

Samsung C48x/C480 계열 USB 복합기를 Ubuntu 홈서버에 연결해, 같은 LAN의
macOS와 Windows에서 프린터와 스캐너로 쓰기 위한 프로젝트입니다.

지원하는 구성은 다음과 같습니다.

- Linux host: Samsung C48x/C480 복합기를 USB로 연결
- Print: CUPS를 IPP로 공유하고 Avahi/Bonjour로 검색
- Scan: SANE Samsung backend를 AirSane eSCL/AirScan으로 공유
- macOS: 기본 프린터 추가, Image Capture 또는 Preview 스캔
- Windows: 기본 네트워크 프린터 추가, NAPS2의 `ESCL Driver` 스캔

Windows 스캔은 NAPS2 eSCL 경로를 표준으로 둡니다. Windows 기본 Scan 앱은 이
AirSane 스캐너를 찾지 못할 수 있습니다. Samsung Universal Scan Driver는 Windows
PC에 USB로 직접 연결된 스캐너용입니다.

## 준비물

- `sudo`를 사용할 수 있는 Ubuntu/Debian 계열 Linux host
- Samsung C48x/C480 계열 USB 복합기
- 같은 LAN에 있는 macOS/Windows 클라이언트
- Windows 스캔용 NAPS2
- host에 Samsung `smfp` SANE backend가 없다면 신뢰할 수 있는 Samsung/SULDR
  scanner backend `.deb`
- AirSane build는 기본으로 프로젝트가 승인한 upstream pin을 사용합니다:
  `SimulPiscator/AirSane` tag `v0.4.12`, commit
  `129cc3bf7258251a0a694dee7741285b59d88f9f`

프린터 전원이 꺼져 있거나 USB가 빠져 있으면 서비스가 떠 있어도 실제 인쇄와
스캔은 실패할 수 있습니다. 먼저 전원과 USB 연결을 확인하세요.

## 빠른 시작

Linux host에서 저장소를 받습니다.

```bash
git clone https://github.com/snowykr/c48x-airbridge.git
cd c48x-airbridge
```

host를 변경하지 않고 bootstrap/build 실행 경로를 먼저 봅니다.

```bash
./scripts/bootstrap-setup.sh --dry-run --no-input
```

guided host setup을 실행합니다.

```bash
./scripts/bootstrap-setup.sh --yes
```

Samsung scanner backend 때문에 `BLOCKED_DRIVER_REQUIRED`가 나오면, 신뢰할 수
있는 로컬 Samsung/SULDR driver package를 지정해서 다시 실행합니다.

```bash
./scripts/bootstrap-setup.sh --yes \
  --suldr-deb /path/to/suld-driver2.deb
```

일반 setup은 AirSane source를 floating branch나 tag에서 가져오지 않습니다. 기본
source는 승인된 upstream tag `v0.4.12`와 commit
`129cc3bf7258251a0a694dee7741285b59d88f9f`입니다. 고급 사용자는
`--airsane-commit <40-character-AirSane-commit>`으로 pin을 바꿀 수 있지만,
branch, tag, `latest` 값은 거부됩니다.

bootstrap script는 Go/build tooling을 확인하고, `sudo go run` 없이 CLI를 빌드한
뒤 `c48x-airbridge setup`을 실행합니다. Go가 없을 때 dry-run/no-input 모드는
host를 바꾸지 않고 필요한 `apt-get` 명령을 그대로 출력합니다.

Make target으로도 같은 진입점을 사용할 수 있습니다.

```bash
make setup-dry-run
make setup
```

`setup`은 privileged 작업 전에 review/apply 경계를 둡니다. 실패를 추측하지 않고
아래 상태 중 하나로 멈춥니다.

- `PASS`: 선택한 host check가 통과했습니다.
- `BLOCKED_PRINTER_REQUIRED`: 프린터 전원이나 USB 연결을 확인하고 다시 실행합니다.
- `BLOCKED_DRIVER_REQUIRED`: Samsung scanner backend가 없으면 신뢰할 수 있는
  Samsung/SULDR `.deb`를 제공합니다. 출력이 AirSane override 문제를 지적하면
  40자 commit으로 바꿔서 다시 실행합니다.
- `BLOCKED_CLIENT_PROOF`: host 준비가 끝났습니다. macOS/Windows 클라이언트에서
  인쇄와 스캔을 확인합니다.
- `FAIL`: 원인을 고친 뒤 setup 또는 verify를 다시 실행합니다.

## Host 검증

비파괴 진단 요약:

```bash
./bin/c48x-airbridge diagnose
```

구조화된 host verification bundle 생성:

```bash
./bin/c48x-airbridge verify --live --output ./host-verify.json
```

host check가 통과하면 `verify`가 macOS와 Windows client handoff를 출력합니다.
클라이언트 등록은 각 기기에서 직접 해야 하므로 자동화하지 않습니다.

## macOS 클라이언트 설정

### 프린트

1. System Settings를 엽니다.
2. Printers & Scanners로 이동합니다.
3. Bonjour shared `Samsung C48x Series @ <host>` 프린터를 추가합니다.
4. test page나 실제 문서를 출력합니다.

### 스캔

1. Image Capture 또는 Preview를 엽니다.
2. AirSane/AirScan으로 광고되는 Samsung C48x scanner를 선택합니다.
3. Preview 또는 Scan을 실행합니다.

macOS는 기본 프린트/스캔 앱을 사용합니다.

## Windows 클라이언트 설정

### 프린트

1. Settings를 엽니다.
2. Bluetooth & devices, Printers & scanners로 이동합니다.
3. `Samsung C48x Series @ <host>` 또는 비슷한 host 이름의 네트워크 프린터를
   추가합니다.
4. test page를 출력합니다.

### 스캔

NAPS2의 eSCL driver를 사용합니다.

1. NAPS2를 설치합니다.
2. Profile을 새로 만들거나 수정합니다.
3. Driver를 `ESCL Driver`로 설정합니다.
4. Samsung C48x/AirSane scanner를 선택합니다.
5. 용지 공급, 크기, 해상도, 색상 설정을 고릅니다.
6. Scan을 실행합니다.

Windows 기본 Scan 앱에서 스캐너가 보이지 않아도 이 구성에서는 이상하지 않습니다.
스캐너는 Windows PC가 아니라 Linux host에 USB로 연결되어 있습니다.

## 자주 쓰는 명령

```bash
./bin/c48x-airbridge help
./bin/c48x-airbridge setup --help
./bin/c48x-airbridge setup --dry-run
./bin/c48x-airbridge verify --live --output ./host-verify.json
make check
```

## 문제 해결

### 클라이언트에서 프린터나 스캐너가 안 보일 때

- 복합기 전원과 USB 연결을 확인합니다.
- host와 클라이언트가 같은 LAN에 있는지 확인합니다.
- Avahi를 확인합니다.

```bash
systemctl status avahi-daemon --no-pager
avahi-browse -rt _ipp._tcp
avahi-browse -rt _uscan._tcp
```

### Host에서는 보이는데 클라이언트에서 안 보일 때

- 방화벽에서 CUPS/IPP와 AirSane/eSCL 접근이 막히지 않았는지 확인합니다.
- 클라이언트에서 다음 URL이 응답하는지 확인합니다.

```bash
curl http://<host>:8090/eSCL/ScannerStatus
```

- Windows는 NAPS2 `ESCL Driver` profile로 확인합니다.

### `BLOCKED_DRIVER_REQUIRED`

installer가 안전한 Samsung scanner backend를 찾지 못했거나, 명시한 AirSane
override가 40자 commit이 아닙니다. 출력에 나온 옵션을 추가해서 다시 실행합니다.

Samsung scanner backend:

```bash
./scripts/bootstrap-setup.sh --yes \
  --suldr-deb /path/to/suld-driver2.deb
```

AirSane 고급 override:

```bash
./scripts/bootstrap-setup.sh --yes \
  --airsane-commit <40-character-AirSane-commit>
```

일반 setup에는 `--airsane-commit`이 필요하지 않습니다. 기본값은 승인된 upstream
pin `129cc3bf7258251a0a694dee7741285b59d88f9f`입니다.

### 인쇄는 되는데 스캔만 안 될 때

프린트와 스캔은 별도 host 경로입니다. 프린트 성공이 SANE/AirSane 성공을 뜻하지는
않습니다.

```bash
scanimage -L
sudo -u saned scanimage -L
curl -f http://localhost:8090/eSCL/ScannerStatus
avahi-browse -rt _uscan._tcp
```

### 수동 fallback

아래 스크립트는 guided setup 출력이 특정 component 문제를 알려줬을 때
troubleshooting이나 targeted repair 용도로만 사용합니다.

```bash
sudo ./scripts/install-cups.sh
sudo ./scripts/install-sane-samsung.sh
sudo AIRSANE_ALLOW_HOST_INSTALL=1 \
  AIRSANE_COMMIT=<40-character-AirSane-commit> \
  ./scripts/install-airsane.sh
```

## 파일 구성

- `bin/c48x-airbridge`: CLI entrypoint
- `cmd/c48x-airbridge/`: Go CLI main
- `internal/cli/`: CLI 명령 구현
- `scripts/bootstrap-setup.sh`: one-command host setup entrypoint
- `scripts/install-cups.sh`: CUPS/Avahi repair helper
- `scripts/install-sane-samsung.sh`: SANE/Samsung scanner backend repair helper
- `scripts/install-airsane.sh`: pinned AirSane build/install repair helper
- `scripts/diagnose.sh`: legacy 비파괴 host 진단
- `configs/udev/99-samsung-c480-scanner.rules`: Samsung USB scanner 권한 rule
- `configs/airsane/access.conf.example`: AirSane 접근 제한 예시
- `testdata/`: CLI test fixture
