# HTTP Request Log

บริการ Go สำหรับรับ HTTP request แล้วเก็บ method, URL, headers, body และเวลาไว้ใน SQLite เหมาะสำหรับตรวจสอบ webhook หรือรับ log จาก API gateway เช่น Kong `http-log`

## เริ่มใช้งานด้วย Docker

```sh
git clone git@github.com:Thirachart/http-request-log.git
cd http-request-log
docker build -t http-request-log .
docker volume create http-request-log-data
docker run --rm --name http-request-log -p 127.0.0.1:8000:8000 --mount source=http-request-log-data,target=/data -w /data http-request-log /root/main
```

ฐานข้อมูลชื่อ `sql_app.db` จะถูกสร้างใน working directory ของโปรเซส ตัวอย่างนี้เก็บไว้ใน Docker volume เพื่อให้ข้อมูลอยู่ต่อหลังลบ container

## รันด้วย Go

ต้องมี Go 1.21 ขึ้นไป และ C compiler ที่ใช้กับ CGO ได้ เนื่องจากใช้ `github.com/mattn/go-sqlite3` หากใช้ Windows และยังไม่ได้ตั้งค่า compiler ให้ใช้ Docker

```sh
go mod download
go run -mod=mod .
```

ต้องเปิด CGO (`CGO_ENABLED=1`) บริการฟังที่ port `8000` ซึ่งกำหนดตายตัวในโค้ด แม้จะโหลด `.env` แต่ยังไม่มี environment variable สำหรับเปลี่ยน port หรือที่เก็บฐานข้อมูล

## Endpoints

| Endpoint | การทำงาน |
| --- | --- |
| `/ingest/` และ path ย่อย เช่น `/ingest/orders` | บันทึก request แล้วตอบ `202 Accepted` พร้อม JSON |
| `/admin/logs` | คืน JSON ของ 50 รายการล่าสุด เรียงตามเวลาจากใหม่ไปเก่า |
| `/` | แสดงข้อความแนะนำการใช้งาน |

Handlers ยังไม่ได้จำกัด HTTP method; ตัวอย่างใช้ POST สำหรับส่งข้อมูลและ GET สำหรับอ่าน log

```sh
curl -i -X POST http://localhost:8000/ingest/demo -H "Content-Type: text/plain" --data "hello"
curl http://localhost:8000/admin/logs
```

แต่ละรายการมี `id`, `method`, `url`, `headers`, `body`, `timestamp` โดย `headers` เป็น string ที่บรรจุ JSON และ `body` เป็น string หากยังไม่มีรายการ endpoint จะคืน `null`

## ใช้กับ Kong

เพิ่ม plugin ให้ service หรือ route ที่ต้องการเก็บ log โดย hostname ต้องเข้าถึงได้จาก Kong:

```yaml
plugins:
  - name: http-log
    config:
      http_endpoint: http://http-request-logger/ingest/
```

ค่า method, URL และ headers ที่ logger เก็บเป็นข้อมูลของ request ที่ Kong ส่งเข้า logger ส่วนรายละเอียด traffic ต้นฉบับอยู่ใน body ที่ Kong ส่งมา ตัวบริการไม่ได้แยกหรือแปลง log payload นั้น

## Kubernetes

ไฟล์ `k8s-deployment.yaml` มี Deployment หนึ่ง replica และ ClusterIP Service port `80` ไปยัง container port `8000` ใน namespace `ole-app1`

```sh
kubectl create namespace ole-app1
kubectl apply -f k8s-deployment.yaml
```

ข้ามการสร้าง namespace หากมีอยู่แล้ว Manifest อ้าง image `snowbee/http-request-logger:latest`; เปลี่ยนเป็น image ที่คุณ build และ push หากต้องการใช้โค้ดจาก checkout นี้ Manifest ยังไม่มี persistent volume จึงไม่รับประกันว่า log จะอยู่ต่อเมื่อ container ถูกสร้างใหม่

## ข้อจำกัด

- ไม่มี authentication สำหรับรับหรืออ่าน log และไม่ปิดบัง token, cookies หรือข้อมูลใน body จึงควรจำกัดการเข้าถึงเฉพาะสภาพแวดล้อมทดสอบที่ควบคุมได้
- อ่าน body ทั้งหมดเข้า memory โดยยังไม่มีขนาดสูงสุด
- ไม่มี pagination, retention policy หรือการลบ log อัตโนมัติ การแสดงเพียง 50 รายการไม่ได้จำกัดจำนวนแถวในฐานข้อมูล
- ไม่ใช่ reverse proxy: รับและบันทึก request แต่ไม่ส่งต่อไป backend

## โครงสร้างและคำสั่งพัฒนา

| ไฟล์ | หน้าที่ |
| --- | --- |
| `main.go` | HTTP handlers และ SQLite storage |
| `go.mod` | Go module และ dependencies |
| `Dockerfile` | Build ด้วย CGO แล้วรันบน Alpine |
| `k8s-deployment.yaml` | ตัวอย่าง Deployment และ Service |

```sh
go build -mod=mod -o http-request-log .
go vet -mod=mod ./...
```

ปัจจุบันยังไม่มี automated tests ก่อนส่งการแก้ไขให้ตรวจการส่ง request และอ่านกลับผ่าน endpoints ข้างต้น และอย่า commit ฐานข้อมูลหรือข้อมูลลับ
