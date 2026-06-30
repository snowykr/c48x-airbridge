# local-printer-scanner

Samsung C480 계열 USB 복합기를 Ubuntu 홈서버에 연결해 LAN 프린터/스캐너로 공유하기 위한 host-native 설치 가이드와 운영 스크립트입니다.

## 공유 대화에서 정리한 핵심 요구사항

- 프린트는 CUPS + Avahi로 공유한다.
- 스캔은 먼저 Ubuntu 서버에서 SANE 장치로 안정적으로 잡혀야 한다.
- C480 계열은 Samsung Unified Linux Driver/SULDR의 `smfp` SANE backend가 필요할 가능성이 높다.
- macOS/Windows 네이티브 스캔 UX는 AirSane으로 eSCL/AirScan 장치를 광고한다.
- macOS/Windows 클라이언트 성공은 사용자가 제공한 수동 증거가 있어야 PASS로 기록한다.
- v1은 클라이언트에 제조사 드라이버 설치를 요구하지 않는다. macOS Image Capture/Preview와 Windows 기본 Scanner 앱으로 검증한다.
- `ipp-usb`는 진단 후보지만 C480 기본 경로의 1순위는 아니다. 성공하면 단순 경로로 전환하고, 실패하면 Samsung ULD/SANE 경로로 간다.

## 기술 스택

| 영역 | 선택 |
| --- | --- |
| OS | Ubuntu/Debian 계열 홈서버, x86_64 권장 |
| 프린트 서버 | CUPS, Avahi/mDNS |
| 스캔 backend | SANE, Samsung ULD/SULDR `smfp` backend |
| 네이티브 스캔 노출 | AirSane, eSCL/AirScan, mDNS |
| 운영 자동화 | Bash 스크립트, systemd 템플릿 |

## 구현 단계

1. 장치/프로토콜 진단
   - USB vendor/product 확인
   - `ipp-usb` 가능성 확인
   - `scanimage -L` / `sudo scanimage -L` / `sudo -u saned scanimage -L` 비교
2. SANE 로컬 스캔 성립
   - SANE 도구 설치
   - Samsung/SULDR 드라이버 설치
   - udev 권한 보정
   - `smfp` backend 확인
3. CUPS 프린트 공유
   - CUPS/Avahi 설치
   - printer sharing 활성화
   - remote admin / unrestricted remote access는 기본 활성화하지 않음
   - test page 출력
4. AirSane 네이티브 스캔
   - AirSane 빌드/설치는 명시 opt-in과 pinned commit이 있을 때만 수행
   - mDNS 광고 확인
   - macOS Image Capture / Windows Scanner 앱에서 테스트
5. macOS/Windows 수동 증거 기록
   - macOS: Image Capture/Preview에서 스캐너 표시와 실제 스캔 결과 확인
   - Windows: 기본 Scanner 앱에서 장치 추가와 실제 스캔 결과 확인
   - 수동 증거가 없으면 `PENDING_MANUAL_QA` 또는 `BLOCKED_PENDING_MANUAL_EVIDENCE` 상태를 유지

## 빠른 사용법

```bash
# 비파괴 진단
./bin/local-printer-scanner diagnose

# 실행될 명령만 확인
./bin/local-printer-scanner install --dry-run
```

이 저장소의 `install` 명령은 repository QA에서 planning-only로 동작합니다. 실제
host 변경은 수행하지 않으며, 설치가 필요하면 `--dry-run` 출력과 `scripts/`,
`configs/`, `systemd/` 파일을 검토한 뒤 Ubuntu 서버에서 수동 운영 절차로
적용합니다.

Host 스크립트를 직접 실행할 때도 안전 기본값을 유지합니다. `scripts/install-cups.sh`는
printer sharing만 활성화하고 remote CUPS admin과 unrestricted remote access는 켜지
않습니다. `scripts/install-airsane.sh`는 기본적으로 clone/build/install을 거부하며,
실제 설치에는 `AIRSANE_ALLOW_HOST_INSTALL=1`과 40자 git commit인
`AIRSANE_COMMIT`이 필요합니다.

## 검증 방법

### 공통 진단

```bash
./bin/local-printer-scanner diagnose
```

성공 기준:

- `lsusb`에서 Samsung `04e8:*` 장치가 보인다.
- `scanimage -L` 또는 `sudo scanimage -L`에서 C480 scanner가 보인다.
- `sudo -u saned scanimage -L`에서도 같은 장치가 보이면 AirSane 권한 조건이 충족된다.

### 프린트

```bash
lpstat -t
lpinfo -v
cupsctl
```

클라이언트 검증:

- macOS: Bonjour/CUPS 공유 프린터가 표시되고 test page 출력 가능
- Windows: 네트워크 프린터 추가 후 test page 출력 가능
- 제조사 클라이언트 드라이버 설치 없이 OS 기본 프린트 경로로 검증

### AirSane

```bash
systemctl status airsane --no-pager
avahi-browse -rt _uscan._tcp
curl -f http://localhost:8090/
```

클라이언트 검증:

- macOS: Image Capture/Preview에서 스캐너 표시
- Windows 10/11: Settings → Bluetooth & devices → Printers & Scanners → Add device 후 Microsoft Scanner 앱에서 스캔
- 제조사 클라이언트 드라이버 설치 없이 OS 기본 스캔 앱으로 검증
- 위 클라이언트 증거가 제출되기 전까지 자동 검증은 PASS가 아니라 `PENDING_MANUAL_QA` 또는 `BLOCKED_PENDING_MANUAL_EVIDENCE`를 반환해야 한다.

### 수동 증거 체크리스트

PASS로 기록하려면 다음 증거를 사용자에게 받아야 합니다.

- macOS 프린트: 공유 프린터 선택 화면과 test page 출력 성공
- macOS 스캔: Image Capture 또는 Preview에서 C480 장치 표시와 실제 스캔 성공
- Windows 프린트: 기본 네트워크 프린터 추가와 test page 출력 성공
- Windows 스캔: 기본 Scanner 앱에서 장치 표시와 실제 스캔 성공
- 실패한 경우: 사용한 OS 버전, 앱 이름, 화면의 오류 문구, 서버에서 같은 시각의 CUPS/AirSane 상태

## 파일 구성

- `bin/local-printer-scanner`: 운영 CLI entrypoint
- `scripts/diagnose.sh`: 비파괴 진단
- `scripts/install-cups.sh`: CUPS/Avahi 설치와 공유 설정
- `scripts/install-sane-samsung.sh`: SANE/Samsung scanner backend 준비
- `scripts/install-airsane.sh`: AirSane build/install 보조
- `configs/udev/99-samsung-c480-scanner.rules`: Samsung USB scanner 권한 rule
- `configs/airsane/access.conf.example`: AirSane 접근 제한 예시

## v1에서 제외한 deferred/archive 항목

다음 항목은 v1 active path가 아닙니다. 설치 명령, systemd 활성화, 문서의 성공 기준에 포함하지 않습니다.

- scanservjs 웹 UI
- Paperless 연동
- OCR 처리
- Docker 배포
- Linux 클라이언트 지원
- Wi-Fi/network-model 흐름
- generic multi-model abstraction
- custom eSCL server

과거 보조 파일은 `scripts/deferred/`와 `systemd/deferred/` 아래에 archive 메모로만 남깁니다. v1 설치와 검증은 CUPS/Avahi, SANE/Samsung, AirSane, macOS/Windows 수동 증거만 다룹니다.
