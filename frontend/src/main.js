import './style.css';
import { EventsOff, EventsOn } from '../wailsjs/runtime/runtime';
import {
  GetConfig,
  IsRunning,
  OpenPath,
  SaveConfig,
  StartMonitoring,
  StopMonitoring,
} from '../wailsjs/go/main/App';

const TEXT = {
  eyebrow: '\u0057\u0069\u006e\u0064\u006f\u0077\u0073\u0020\u004e\u0065\u0074\u0077\u006f\u0072\u006b\u0020\u004d\u006f\u006e\u0069\u0074\u006f\u0072',
  subtitle: '\u6301\u7eed\u8bb0\u5f55 RTT\u3001\u603b\u8017\u65f6\u3001\u4e22\u5305\u548c\u8fde\u7eed\u8d85\u65f6\uff0c\u9002\u5408\u6392\u67e5\u5bbd\u5e26\u3001DNS \u6216\u94fe\u8def\u6296\u52a8\u3002',
  runStateIdle: '\u672a\u542f\u52a8',
  runStateRunning: '\u76d1\u63a7\u4e2d',
  panelConfig: '\u76d1\u63a7\u914d\u7f6e',
  panelConfigDesc: '\u5e38\u7528\u914d\u7f6e\u90fd\u53ef\u4ee5\u76f4\u63a5\u4fee\u6539\u5e76\u4fdd\u5b58\u5230\u672c\u5730\u914d\u7f6e\u6587\u4ef6\u3002',
  host: '\u76ee\u6807\u4e3b\u673a',
  interval: '\u63a2\u6d4b\u95f4\u9694',
  timeout: '\u8d85\u65f6\u65f6\u95f4',
  logDir: '\u65e5\u5fd7\u76ee\u5f55',
  reportDir: '\u65e5\u62a5\u76ee\u5f55',
  summaryEvery: '\u6c47\u603b\u95f4\u9694',
  reportEvery: '\u65e5\u62a5\u5237\u65b0',
  placeholderHost: '\u5982\uff1awww.baidu.com / 1.1.1.1',
  placeholderInterval: '\u5982\uff1a1s',
  placeholderTimeout: '\u5982\uff1a2s',
  placeholderLogs: '\u5982\uff1alogs',
  placeholderReports: '\u5982\uff1areports',
  placeholder30s: '\u5982\uff1a30s',
  saveConfig: '\u4fdd\u5b58\u914d\u7f6e',
  startMonitoring: '\u542f\u52a8\u76d1\u63a7',
  stopMonitoring: '\u505c\u6b62\u76d1\u63a7',
  panelStatus: '\u5b9e\u65f6\u72b6\u6001',
  panelStatusDesc: '\u8fd9\u91cc\u663e\u793a\u5f53\u524d\u7f51\u7edc\u8d28\u91cf\u548c\u5f53\u5929\u7d2f\u8ba1\u7edf\u8ba1\u3002',
  metricExplain: '\u8ba1\u6570\u8bf4\u660e\uff1a\u6210\u529f\u8868\u793a\u6536\u5230\u56de\u590d\uff0c\u4e22\u5305\u8868\u793a\u6240\u6709\u5931\u8d25\u63a2\u6d4b\uff0c\u8d85\u65f6\u662f\u5931\u8d25\u4e2d\u7684\u8d85\u65f6\u90e8\u5206\u3002',
  currentStatus: '\u5f53\u524d\u72b6\u6001',
  rttTotal: 'RTT / Total',
  lossRate: '\u4e22\u5305\u7387',
  count: '\u7d2f\u8ba1\u7edf\u8ba1',
  maxTimeout: '\u6700\u5927\u8fde\u7eed\u8d85\u65f6',
  reportFile: '\u65e5\u62a5\u6587\u4ef6',
  openLogs: '\u6253\u5f00\u65e5\u5fd7\u76ee\u5f55',
  openReports: '\u6253\u5f00\u65e5\u62a5\u76ee\u5f55',
  panelLog: '\u4e8b\u4ef6\u65e5\u5fd7',
  panelLogDesc: '\u6309\u65f6\u95f4\u987a\u5e8f\u663e\u793a\u76d1\u63a7\u4e8b\u4ef6\uff0c\u65b9\u4fbf\u5feb\u901f\u5b9a\u4f4d\u5f02\u5e38\u4e0e\u6062\u590d\u3002',
  waitingEvent: '\u7b49\u5f85\u4e8b\u4ef6...',
  started: '\u542f\u52a8\u4e2d',
  stopped: '\u5df2\u505c\u6b62',
  monitorStarted: '\u76d1\u63a7\u5df2\u542f\u52a8\u3002',
  configSaved: '\u914d\u7f6e\u5df2\u4fdd\u5b58\u3002',
  lastEvent: '\u6700\u540e\u4e8b\u4ef6\uff1a',
  success: '\u6210\u529f',
  loss: '\u4e22\u5305',
  timeoutShort: '\u8d85\u65f6',
};

const app = document.querySelector('#app');

app.innerHTML = `
  <div class="shell">
    <section class="hero">
      <div class="hero-copy">
        <p class="eyebrow">${TEXT.eyebrow}</p>
        <h1>NetSentry</h1>
        <p class="subtitle">${TEXT.subtitle}</p>
      </div>
      <div class="hero-status">
        <span class="hero-status-label">${TEXT.currentStatus}</span>
        <strong class="hero-stat" id="runState">${TEXT.runStateIdle}</strong>
      </div>
    </section>

    <section class="dashboard-grid">
      <div class="panel panel-config">
        <div class="panel-head">
          <div>
            <h2>${TEXT.panelConfig}</h2>
            <p class="panel-desc">${TEXT.panelConfigDesc}</p>
          </div>
        </div>

        <div class="form-grid">
          <div class="field">
            <label for="host">${TEXT.host}</label>
            <input id="host" placeholder="${TEXT.placeholderHost}" />
          </div>
          <div class="field">
            <label for="interval">${TEXT.interval}</label>
            <input id="interval" placeholder="${TEXT.placeholderInterval}" />
          </div>
          <div class="field">
            <label for="timeout">${TEXT.timeout}</label>
            <input id="timeout" placeholder="${TEXT.placeholderTimeout}" />
          </div>
          <div class="field field-wide">
            <label for="logDir">${TEXT.logDir}</label>
            <input id="logDir" placeholder="${TEXT.placeholderLogs}" />
          </div>
          <div class="field field-wide">
            <label for="reportDir">${TEXT.reportDir}</label>
            <input id="reportDir" placeholder="${TEXT.placeholderReports}" />
          </div>
          <div class="field">
            <label for="summaryEvery">${TEXT.summaryEvery}</label>
            <input id="summaryEvery" placeholder="${TEXT.placeholder30s}" />
          </div>
          <div class="field">
            <label for="reportEvery">${TEXT.reportEvery}</label>
            <input id="reportEvery" placeholder="${TEXT.placeholder30s}" />
          </div>
        </div>

        <div class="button-row action-row">
          <button id="saveBtn" class="ghost">${TEXT.saveConfig}</button>
          <button id="startBtn" class="primary">${TEXT.startMonitoring}</button>
          <button id="stopBtn" class="danger">${TEXT.stopMonitoring}</button>
        </div>
      </div>

      <div class="panel panel-status">
        <div class="panel-head stacked">
          <div>
            <h2>${TEXT.panelStatus}</h2>
            <p class="panel-desc">${TEXT.panelStatusDesc}</p>
          </div>
          <p class="metric-explain">${TEXT.metricExplain}</p>
        </div>

        <div class="metrics-grid">
          <div class="metric-card accent">
            <span>${TEXT.currentStatus}</span>
            <div class="status-badge idle" id="statusBadge">
              <strong id="statusText">${TEXT.runStateIdle}</strong>
            </div>
          </div>
          <div class="metric-card">
            <span>${TEXT.rttTotal}</span>
            <strong id="latencyText">- / -</strong>
          </div>
          <div class="metric-card">
            <span>${TEXT.lossRate}</span>
            <strong id="lossText">0.00%</strong>
          </div>
          <div class="metric-card">
            <span>${TEXT.count}</span>
            <strong id="countText">${TEXT.success} 0 / ${TEXT.loss} 0 / ${TEXT.timeoutShort} 0</strong>
          </div>
          <div class="metric-card">
            <span>${TEXT.maxTimeout}</span>
            <strong id="streakText">0</strong>
          </div>
          <div class="metric-card metric-path">
            <span>${TEXT.reportFile}</span>
            <strong id="reportPath">-</strong>
          </div>
        </div>

        <div class="button-row compact">
          <button id="openLogsBtn" class="ghost">${TEXT.openLogs}</button>
          <button id="openReportsBtn" class="ghost">${TEXT.openReports}</button>
        </div>
      </div>
    </section>

    <section class="panel log-panel">
      <div class="panel-head">
        <div>
          <h2>${TEXT.panelLog}</h2>
          <p class="panel-desc">${TEXT.panelLogDesc}</p>
        </div>
        <span id="lastEventHint">${TEXT.waitingEvent}</span>
      </div>
      <textarea id="eventLog" class="event-console" readonly spellcheck="false"></textarea>
    </section>
  </div>
`;

const fields = {
  host: document.getElementById('host'),
  interval: document.getElementById('interval'),
  timeout: document.getElementById('timeout'),
  log_dir: document.getElementById('logDir'),
  report_dir: document.getElementById('reportDir'),
  summary_every: document.getElementById('summaryEvery'),
  report_every: document.getElementById('reportEvery'),
};

const runState = document.getElementById('runState');
const statusBadge = document.getElementById('statusBadge');
const statusText = document.getElementById('statusText');
const latencyText = document.getElementById('latencyText');
const lossText = document.getElementById('lossText');
const countText = document.getElementById('countText');
const streakText = document.getElementById('streakText');
const reportPath = document.getElementById('reportPath');
const eventLog = document.getElementById('eventLog');
const lastEventHint = document.getElementById('lastEventHint');
const startBtn = document.getElementById('startBtn');
const stopBtn = document.getElementById('stopBtn');
const saveBtn = document.getElementById('saveBtn');
const openLogsBtn = document.getElementById('openLogsBtn');
const openReportsBtn = document.getElementById('openReportsBtn');
const MAX_LOG_ENTRIES = 500;
const logLines = [];

function cfgFromForm() {
  return {
    host: fields.host.value.trim(),
    interval: fields.interval.value.trim(),
    timeout: fields.timeout.value.trim(),
    log_dir: fields.log_dir.value.trim(),
    report_dir: fields.report_dir.value.trim(),
    summary_every: fields.summary_every.value.trim(),
    report_every: fields.report_every.value.trim(),
  };
}

function setForm(cfg) {
  fields.host.value = cfg.host ?? '';
  fields.interval.value = cfg.interval ?? '';
  fields.timeout.value = cfg.timeout ?? '';
  fields.log_dir.value = cfg.log_dir ?? '';
  fields.report_dir.value = cfg.report_dir ?? '';
  fields.summary_every.value = cfg.summary_every ?? '';
  fields.report_every.value = cfg.report_every ?? '';
}

function setRunning(running) {
  runState.textContent = running ? TEXT.runStateRunning : TEXT.runStateIdle;
  startBtn.disabled = running;
  stopBtn.disabled = !running;
}

function setStatus(text, tone) {
  statusText.textContent = text;
  statusBadge.className = `status-badge ${tone}`;
}

function appendLog(line) {
  logLines.push(line);
  if (logLines.length > MAX_LOG_ENTRIES) {
    logLines.splice(0, logLines.length - MAX_LOG_ENTRIES);
  }
  eventLog.value = logLines.join('\n');
  eventLog.scrollTop = eventLog.scrollHeight;
  lastEventHint.textContent = `${TEXT.lastEvent}${new Date().toLocaleTimeString()}`;
}

function updateCountText(payload) {
  const lost = payload.timeout + payload.unreachable + payload.errors;
  countText.textContent = `${TEXT.success} ${payload.success} / ${TEXT.loss} ${lost} / ${TEXT.timeoutShort} ${payload.timeout}`;
}

async function init() {
  const cfg = await GetConfig();
  setForm(cfg);
  setRunning(await IsRunning());
  setStatus(TEXT.runStateIdle, 'idle');
}

saveBtn.addEventListener('click', async () => {
  try {
    await SaveConfig(cfgFromForm());
    appendLog(TEXT.configSaved);
  } catch (error) {
    appendLog(`[error] ${error}`);
  }
});

startBtn.addEventListener('click', async () => {
  try {
    logLines.length = 0;
    eventLog.value = '';
    await StartMonitoring(cfgFromForm());
    setRunning(true);
  } catch (error) {
    appendLog(`[error] ${error}`);
  }
});

stopBtn.addEventListener('click', async () => {
  try {
    await StopMonitoring();
  } catch (error) {
    appendLog(`[error] ${error}`);
  }
});

openLogsBtn.addEventListener('click', async () => {
  try {
    await OpenPath(fields.log_dir.value || 'logs');
  } catch (error) {
    appendLog(`[error] ${error}`);
  }
});

openReportsBtn.addEventListener('click', async () => {
  try {
    await OpenPath(fields.report_dir.value || 'reports');
  } catch (error) {
    appendLog(`[error] ${error}`);
  }
});

EventsOn('monitor:started', (payload) => {
  setRunning(true);
  reportPath.textContent = payload.reportPath || '-';
  setStatus(TEXT.started, 'summary');
  appendLog(TEXT.monitorStarted);
});

EventsOn('monitor:probe', (payload) => {
  const status = payload.status.toUpperCase();
  const toneMap = {
    OK: 'ok',
    TIMEOUT: 'timeout',
    UNREACHABLE: 'unreachable',
    ERROR: 'error',
  };
  setStatus(status, toneMap[status] || 'neutral');
  latencyText.textContent = `${payload.rttMs}ms / ${payload.totalMs}ms`;
  lossText.textContent = `${payload.lossRate.toFixed(2)}%`;
  updateCountText(payload);
  streakText.textContent = `${payload.maxTimeouts}`;
  appendLog(payload.summaryLine);
});

EventsOn('monitor:summary', (payload) => {
  appendLog(payload.summaryLine);
});

EventsOn('monitor:stopped', (payload) => {
  setRunning(false);
  setStatus(TEXT.stopped, 'idle');
  lossText.textContent = `${payload.lossRate.toFixed(2)}%`;
  updateCountText(payload);
  streakText.textContent = `${payload.maxTimeouts}`;
  appendLog(payload.summaryLine.replace('[summary]', '[stopped]'));
});

window.addEventListener('beforeunload', () => {
  EventsOff('monitor:started');
  EventsOff('monitor:probe');
  EventsOff('monitor:summary');
  EventsOff('monitor:stopped');
});

init();
