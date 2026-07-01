# c48x-airbridge

Samsung C48x/C480 계열 USB 복합기를 Ubuntu 홈서버에 연결해, 같은 LAN의
macOS와 Windows에서 프린터와 스캐너로 쓰기 위한 host-native 운영 가이드입니다.

현재 검증된 최종 구성은 다음과 같습니다.

- Linux host: USB로 복합기 연결
- Print: CUPS + Avahi/Bonjour 공유
- Scan: SANE Samsung backend + AirSane eSCL/AirScan 공유
- macOS: 기본 프린터 추가, Image Capture/Preview 스캔
- Windows: 기본 네트워크 프린터 추가, NAPS2의 `ESCL Driver`로 스캔

Windows 기본 스캔 앱이 아니라 NAPS2 eSCL 경로를 표준 스캔 경로로 둡니다. 이
장비가 Linux host에 USB로 연결된 상태에서는 Samsung Universal Scan Driver 같은
Windows 로컬 USB 스캔 드라이버를 쓰는 경로가 아닙니다.

## 준비물

- Ubuntu/Debian 계열 Linux host
- Samsung C48x/C480 계열 USB 복합기
- 같은 LAN에 있는 macOS/Windows 클라이언트
- host에서 `sudo` 가능한 계정
- Windows 스캔용 NAPS2

프린터 전원이 꺼져 있거나 USB가 빠져 있으면 CUPS/AirSane 서비스는 살아 있어도
실제 인쇄와 스캔은 실패할 수 있습니다. 점검할 때는 먼저 복합기 전원과 USB 연결을
확인하세요.

## 저장소 받기

```bash
git clone https://github.com/snowykr/c48x-airbridge.git
cd c48x-airbridge
```

비파괴 점검용 CLI는 Go로 작성되어 있습니다.

```bash
./bin/c48x-airbridge help
./bin/c48x-airbridge diagnose
./bin/c48x-airbridge install --dry-run
```

`install --dry-run`은 실제 설치를 하지 않습니다. 실제 host 변경은 아래 설치
스크립트를 필요한 단계별로 직접 실행합니다.

## Linux Host 초기 설치

### 1. USB 장치 확인

복합기를 켜고 Linux host에 USB로 연결한 뒤 확인합니다.

```bash
lsusb | grep -i samsung
```

Samsung 장치가 보여야 다음 단계로 진행할 수 있습니다.

### 2. CUPS/Avahi 프린트 공유 설치

```bash
sudo ./scripts/install-cups.sh
```

이 스크립트는 CUPS와 Avahi를 설치하고 printer sharing만 켭니다. 원격 CUPS
관리자 기능이나 unrestricted remote access는 켜지 않습니다.

설치 후 CUPS에서 USB 프린터 큐를 추가합니다.

```bash
lpinfo -v
lpstat -t
```

필요하면 브라우저에서 host 로컬로 CUPS 관리 화면을 엽니다.

```text
http://localhost:631/
```

Samsung C48x/C480 USB 프린터를 추가한 뒤 test page가 출력되는지 확인합니다.

### 3. SANE/Samsung 스캔 backend 준비

```bash
sudo ./scripts/install-sane-samsung.sh
```

이 스크립트는 SANE/USB 관련 패키지와 Samsung USB scanner udev rule을 설치하고,
`saned` 계정이 scanner/lp 그룹을 사용할 수 있게 보정합니다.

Samsung `smfp` backend가 아직 없으면 신뢰할 수 있는 SULDR/ULD 패키지를 별도로
설치해야 할 수 있습니다. 로컬 스캔 장치가 잡히는지 세 경로로 확인합니다.

```bash
scanimage -L
sudo scanimage -L
sudo -u saned scanimage -L
```

AirSane까지 안정적으로 쓰려면 `sudo -u saned scanimage -L`에서도 같은 스캐너가
보이는 상태가 가장 좋습니다.

### 4. AirSane 설치

AirSane 설치 스크립트는 기본적으로 실제 clone/build/install을 거부합니다. host에
설치하려면 AirSane upstream commit을 40자 해시로 고정하고 명시적으로 opt-in합니다.

```bash
sudo AIRSANE_ALLOW_HOST_INSTALL=1 \
  AIRSANE_COMMIT=<40-character-git-commit> \
  ./scripts/install-airsane.sh
```

설치 후 확인합니다.

```bash
systemctl status airsaned --no-pager
avahi-browse -rt _uscan._tcp
curl -f http://localhost:8090/
```

`_uscan._tcp` 광고가 보이고 `http://localhost:8090/`가 응답하면 LAN 클라이언트가
eSCL/AirScan 스캐너로 볼 수 있는 상태입니다.

## macOS에서 사용

### 프린트

1. System Settings → Printers & Scanners
2. Add Printer
3. Bonjour Shared로 보이는 `Samsung C48x Series @ <host>` 선택
4. 추가 후 test page 또는 실제 문서 출력

### 스캔

1. Image Capture 또는 Preview 실행
2. AirSane/AirScan으로 보이는 Samsung C48x 스캐너 선택
3. Preview 또는 Scan 실행

macOS는 기본 Image Capture/Preview 경로로 인쇄와 스캔이 모두 동작하는 구성이
검증되어 있습니다.

## Windows에서 사용

### 프린트

1. Settings → Bluetooth & devices → Printers & scanners
2. Add device
3. `Samsung C48x Series @ <host>` 또는 host 이름이 붙은 네트워크 프린터 추가
4. 테스트 페이지 출력

프린트는 Windows 기본 네트워크 프린터 경로로 동작합니다.

### 스캔

Windows 스캔은 NAPS2의 eSCL 경로를 사용합니다.

1. NAPS2 설치
2. Profiles → New 또는 Edit
3. Driver를 `ESCL Driver`로 선택
4. 검색된 Samsung C48x/AirSane 스캐너 선택
5. 용지 공급, 용지 크기, 해상도, 색상 설정 후 Scan

Windows 기본 스캔 앱에서 장치가 보이지 않는 것은 이 구성에서 이상한 상태가
아닙니다. 복합기는 Windows PC에 USB로 연결된 것이 아니므로 Samsung Universal Scan
Driver 같은 로컬 USB 드라이버도 사용 경로가 아닙니다.

## 평소 점검 명령

Linux host에서 프린트 상태:

```bash
lpstat -t
lpinfo -v
avahi-browse -rt _ipp._tcp
```

Linux host에서 스캔 상태:

```bash
scanimage -L
sudo -u saned scanimage -L
systemctl status airsaned --no-pager
avahi-browse -rt _uscan._tcp
curl -f http://localhost:8090/
```

저장소 CLI의 비파괴 점검:

```bash
./bin/c48x-airbridge diagnose
```

## 문제 해결

### 클라이언트에서 프린터나 스캐너가 안 보일 때

- 복합기 전원과 USB 연결 확인
- host와 클라이언트가 같은 LAN에 있는지 확인
- Avahi 서비스 확인:

```bash
systemctl status avahi-daemon --no-pager
avahi-browse -rt _ipp._tcp
avahi-browse -rt _uscan._tcp
```

### Linux host에서는 보이는데 macOS/Windows에서 안 보일 때

- 방화벽에서 CUPS/IPP와 AirSane/eSCL 접근이 막히지 않았는지 확인
- `curl http://<host>:8090/eSCL/ScannerStatus`가 클라이언트에서 응답하는지 확인
- Windows는 기본 스캔 앱 대신 NAPS2 `ESCL Driver` 프로필로 확인

### 스캔이 시작되지 않거나 멈출 때

- 복합기 패널에 오류, 절전, 용지/덮개 상태가 없는지 확인
- 먼저 Linux host에서 로컬 스캔이 되는지 확인:

```bash
scanimage -L
scanimage --format=png --output-file=/tmp/c48x-test.png
```

- 이후 AirSane 서비스를 재시작:

```bash
sudo systemctl restart airsaned
systemctl status airsaned --no-pager
```

### 인쇄는 되는데 스캔만 안 될 때

프린트와 스캔은 별도 경로입니다. 프린트가 된다고 해서 SANE/AirSane 스캔 경로가
정상이라는 뜻은 아닙니다.

```bash
sudo -u saned scanimage -L
curl -f http://localhost:8090/
avahi-browse -rt _uscan._tcp
```

세 명령을 먼저 확인하세요.

## 파일 구성

- `bin/c48x-airbridge`: 운영 CLI entrypoint
- `cmd/c48x-airbridge/`: Go CLI main
- `internal/cli/`: CLI 명령 구현
- `scripts/install-cups.sh`: CUPS/Avahi 설치와 프린터 공유 설정
- `scripts/install-sane-samsung.sh`: SANE/Samsung scanner backend 준비
- `scripts/install-airsane.sh`: AirSane pinned build/install 보조
- `scripts/diagnose.sh`: host 비파괴 진단
- `configs/udev/99-samsung-c480-scanner.rules`: Samsung USB scanner 권한 rule
- `configs/airsane/access.conf.example`: AirSane 접근 제한 예시
- `testdata/`: CLI 테스트 fixture
