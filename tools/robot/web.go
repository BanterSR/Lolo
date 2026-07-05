package main

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
)

// sample 是每个上报周期的一帧快照，供仪表盘绘制时间序列。
type sample struct {
	T            int64   `json:"t"` // unix 毫秒
	Elapsed      float64 `json:"elapsed"`
	Active       int64   `json:"active"`
	Target       int     `json:"target"`
	RPS          float64 `json:"rps"`
	LoginRate    float64 `json:"loginRate"`
	P50          float64 `json:"p50"`
	P95          float64 `json:"p95"`
	P99          float64 `json:"p99"`
	LoginOK      int64   `json:"loginOK"`
	LoginFail    int64   `json:"loginFail"`
	Disconnects  int64   `json:"disconnects"`
	Timeouts     int64   `json:"timeouts"`
	SrvGoroutine int     `json:"srvGoroutine"`
	SrvHeapMB    float64 `json:"srvHeapMB"`
	SrvNumGC     uint32  `json:"srvNumGC"`
	SrvOK        bool    `json:"srvOK"`
}

// History 是有界的样本环形缓冲，供 web 仪表盘读取。
type History struct {
	mu    sync.Mutex
	data  []sample
	maxN  int
	hw    HardwareInfo
	cfg   *Config
	pprof string
}

func newHistory(maxN int, hw HardwareInfo, cfg *Config, pprof string) *History {
	return &History{maxN: maxN, hw: hw, cfg: cfg, pprof: pprof}
}

func (h *History) add(s sample) {
	h.mu.Lock()
	h.data = append(h.data, s)
	if len(h.data) > h.maxN {
		h.data = h.data[len(h.data)-h.maxN:]
	}
	h.mu.Unlock()
}

type statsResponse struct {
	HW       HardwareInfo `json:"hw"`
	Mode     string       `json:"mode"`
	Scenario string       `json:"scenario"`
	Gate     string       `json:"gate"`
	Sdk      string       `json:"sdk"`
	Pprof    string       `json:"pprof"`
	Samples  []sample     `json:"samples"`
}

func (h *History) response() statsResponse {
	h.mu.Lock()
	out := make([]sample, len(h.data))
	copy(out, h.data)
	h.mu.Unlock()
	return statsResponse{
		HW:       h.hw,
		Mode:     h.cfg.Mode,
		Scenario: h.cfg.Scenario,
		Gate:     h.cfg.Gate,
		Sdk:      h.cfg.Sdk,
		Pprof:    h.pprof,
		Samples:  out,
	}
}

// startWeb 启动仪表盘 HTTP 服务并立即返回；调用方负责 Shutdown。
// 先同步 Listen，绑定失败（如端口被占用）会立即返回错误而非被吞掉。
func startWeb(addr string, h *History) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(dashboardHTML))
	})
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(h.response())
	})
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	return srv, nil
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="utf-8">
<title>Lolo 压测仪表盘</title>
<style>
  body{background:#12151b;color:#d7dae0;font:13px/1.5 -apple-system,Segoe UI,Roboto,monospace;margin:0;padding:16px}
  h1{font-size:16px;margin:0 0 4px}
  .meta{color:#8a93a2;font-size:12px;margin-bottom:12px}
  .meta a{color:#4fc3f7;text-decoration:none}
  .grid{display:grid;grid-template-columns:repeat(2,minmax(320px,1fr));gap:12px}
  .card{background:#1a1f28;border:1px solid #262c38;border-radius:8px;padding:10px}
  .card h2{font-size:12px;margin:0 0 6px;color:#9aa4b2;font-weight:600}
  canvas{width:100%;height:150px;display:block}
  .nums{display:grid;grid-template-columns:repeat(auto-fill,minmax(150px,1fr));gap:8px;margin-top:12px}
  .kv{background:#1a1f28;border:1px solid #262c38;border-radius:6px;padding:8px}
  .kv .k{color:#8a93a2;font-size:11px}
  .kv .v{font-size:18px;font-weight:600}
  .warn{color:#e57373}
  .ok{color:#81c784}
</style>
</head>
<body>
  <h1>Lolo 压测仪表盘</h1>
  <div class="meta" id="meta">连接中…</div>
  <div class="grid">
    <div class="card"><h2>并发 CCU（active / target）</h2><canvas id="cccu"></canvas></div>
    <div class="card"><h2>吞吐（rps / 登录·秒⁻¹）</h2><canvas id="cthr"></canvas></div>
    <div class="card"><h2>RTT 延迟 ms（p50 / p95 / p99）</h2><canvas id="clat"></canvas></div>
    <div class="card"><h2>服务端 goroutine</h2><canvas id="csrv"></canvas></div>
  </div>
  <div class="nums" id="nums"></div>
<script>
function draw(id, series){
  const c=document.getElementById(id);
  const dpr=window.devicePixelRatio||1;
  const w=c.clientWidth,h=c.clientHeight;
  if(c.width!==w*dpr){c.width=w*dpr;c.height=h*dpr;}
  const ctx=c.getContext('2d');ctx.setTransform(dpr,0,0,dpr,0,0);
  ctx.clearRect(0,0,w,h);
  const pad=30;let maxv=1;
  series.forEach(s=>s.data.forEach(v=>{if(v>maxv)maxv=v;}));
  maxv*=1.15;
  const n=Math.max(...series.map(s=>s.data.length),1);
  ctx.strokeStyle='#2a3240';ctx.lineWidth=1;
  ctx.beginPath();ctx.moveTo(pad,4);ctx.lineTo(pad,h-16);ctx.lineTo(w-4,h-16);ctx.stroke();
  ctx.fillStyle='#6b7280';ctx.font='11px monospace';
  ctx.fillText(maxv.toFixed(maxv<10?1:0),2,12);ctx.fillText('0',pad-14,h-16);
  series.forEach(s=>{
    ctx.strokeStyle=s.color;ctx.lineWidth=1.6;ctx.beginPath();
    s.data.forEach((v,i)=>{
      const x=pad+(w-pad-6)*(n<=1?0:i/(n-1));
      const y=(h-16)-(h-22)*(v/maxv);
      i===0?ctx.moveTo(x,y):ctx.lineTo(x,y);
    });
    ctx.stroke();
  });
  let lx=pad+6;
  series.forEach(s=>{
    ctx.fillStyle=s.color;ctx.fillRect(lx,5,10,3);
    ctx.fillStyle='#aab2c0';ctx.fillText(s.name,lx+14,10);
    lx+=14+ctx.measureText(s.name).width+16;
  });
}
function kv(k,v,cls){return '<div class="kv"><div class="k">'+k+'</div><div class="v '+(cls||'')+'">'+v+'</div></div>';}
async function poll(){
  let d;try{d=await(await fetch('/api/stats')).json();}catch(e){return;}
  const s=d.samples||[];const last=s[s.length-1]||{};
  const hw=d.hw||{};
  document.getElementById('meta').innerHTML=
    '模式 <b>'+d.mode+'</b> / 场景 <b>'+d.scenario+'</b> · 网关 '+d.gate+
    ' · 客户端 '+hw.os+'/'+hw.arch+' '+hw.numCPU+'核(GOMAXPROCS='+hw.gomaxprocs+') '+hw.hostname+' '+hw.goVersion+
    ' · <a href="'+d.pprof+'" target="_blank">服务端 pprof ↗</a>';
  draw('cccu',[{name:'active',color:'#4fc3f7',data:s.map(x=>x.active)},{name:'target',color:'#6b7280',data:s.map(x=>x.target)}]);
  draw('cthr',[{name:'rps',color:'#81c784',data:s.map(x=>x.rps)},{name:'login/s',color:'#ffb74d',data:s.map(x=>x.loginRate)}]);
  draw('clat',[{name:'p50',color:'#4fc3f7',data:s.map(x=>x.p50)},{name:'p95',color:'#ffb74d',data:s.map(x=>x.p95)},{name:'p99',color:'#e57373',data:s.map(x=>x.p99)}]);
  draw('csrv',[{name:'goroutine',color:'#ba68c8',data:s.map(x=>x.srvGoroutine)}]);
  const srv=last.srvOK?(last.srvGoroutine+' / '+last.srvHeapMB.toFixed(1)+'MB / GC'+last.srvNumGC):'n/a';
  document.getElementById('nums').innerHTML=
    kv('运行时间',(last.elapsed||0).toFixed(0)+'s')+
    kv('在线 CCU',last.active||0)+
    kv('登录成功',last.loginOK||0,'ok')+
    kv('登录失败',last.loginFail||0,(last.loginFail?'warn':''))+
    kv('RPS',(last.rps||0).toFixed(0))+
    kv('p95 延迟',(last.p95||0).toFixed(1)+'ms')+
    kv('p99 延迟',(last.p99||0).toFixed(1)+'ms')+
    kv('断线',last.disconnects||0,(last.disconnects?'warn':''))+
    kv('超时',last.timeouts||0,(last.timeouts?'warn':''))+
    kv('服务端 gr/堆/GC',srv,last.srvOK?'':'warn');
}
setInterval(poll,1000);poll();
</script>
</body>
</html>`
