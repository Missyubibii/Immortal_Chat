const API_BASE = "/api";
let currentConversationId = null;
let refreshTimer = null;

// Lấy Secret Key từ URL (Ví dụ: ?secret_key=abc...)
const urlParams = new URLSearchParams(window.location.search);
const MESH_SECRET = urlParams.get('secret_key');

// Hàm tạo Header chuẩn (kèm Secret Key)
function getAuthHeaders() {
  const headers = { "Content-Type": "application/json" };
  if (MESH_SECRET) {
    headers["X-Mesh-Secret"] = MESH_SECRET; // Chìa khóa vạn năng cho Admin
  }
  return headers;
}

document.addEventListener("DOMContentLoaded", () => {

  // Kiểm tra bảo mật
  if (!MESH_SECRET) {
    console.warn("⚠️ Không tìm thấy secret_key trên URL. API có thể bị lỗi 403.");
    alert("Vui lòng truy cập kèm ?secret_key=YOUR_KEY để có quyền Admin");
  }

  // 1. Init UI
  switchView("dashboard");

  // 2. Start Real Monitoring (Giống file gốc)
  loadAllSystemData();
  refreshTimer = setInterval(loadAllSystemData, 5000); // 5s refresh

  // 3. Event Listeners
  setupEventListeners();
});

// --- CORE SYSTEM LOGIC (FROM OLD DASHBOARD.JS) ---

async function loadAllSystemData() {
  try {
    await Promise.all([
      loadSystemMetrics(), // CPU, RAM
      loadSystemStatus(), // Uptime, Version
      loadPlatforms(), // Facebook status
      loadSyncStatus(), // Database sync
    ]);
  } catch (e) {
    console.error("Auto-refresh error:", e);
  }
}

async function loadSystemMetrics() {
  try {
    const res = await fetch(`${API_BASE}/system/metrics`);
    if (!res.ok) return;
    const data = await res.json();

    // Update CPU
    document.getElementById(
      "metric-cpu"
    ).innerText = `${data.cpu_percent.toFixed(1)}%`;
    document.getElementById("bar-cpu").style.width = `${Math.min(
      data.cpu_percent,
      100
    )}%`;

    // Update RAM
    const ramPercent = (data.ram_used_gb / data.ram_total_gb) * 100;
    document.getElementById(
      "metric-ram"
    ).innerText = `${data.ram_used_gb.toFixed(1)}/${data.ram_total_gb.toFixed(
      1
    )} GB`;
    document.getElementById("bar-ram").style.width = `${Math.min(
      ramPercent,
      100
    )}%`;

    // Goroutines
    document.getElementById("metric-goroutines").innerText =
      data.goroutines_count;
  } catch (e) {
    console.warn("Metrics error");
  }
}

async function loadSystemStatus() {
  try {
    const res = await fetch(`${API_BASE}/status`);
    if (!res.ok) return;
    const data = await res.json();

    document.getElementById(
      "system-uptime"
    ).innerText = `Uptime: ${data.uptime}`;
    document.getElementById("stat-version").innerText = data.version;
    document.getElementById(
      "stat-tenant"
    ).innerText = `Tenant: ${data.tenant_id}`;
  } catch (e) {
    console.warn("Status error");
  }
}

async function loadPlatforms() {
  try {
    const res = await fetch(`${API_BASE}/platforms`);
    if (!res.ok) return;
    const data = await res.json(); // Array

    // Tìm Facebook Platform để hiển thị status
    const fb = data.find((p) => p.platform === "facebook");
    const statusEl = document.getElementById("stat-fb-status");
    if (fb) {
      statusEl.innerText = fb.status === "connected" ? "Connected" : "Error";
      statusEl.className = `text-xl font-bold mt-1 ${
        fb.status === "connected" ? "text-green-600" : "text-red-500"
      }`;
    } else {
      statusEl.innerText = "Not Configured";
    }
  } catch (e) {
    console.warn("Platforms error");
  }
}

async function loadSyncStatus() {
  try {
    const res = await fetch(`${API_BASE}/sync/status`);
    if (!res.ok) return;
    const data = await res.json();

    document.getElementById("stat-sync-pending").innerText =
      data.pending_messages;
    const lastSync = new Date(data.last_sync_at);
    const diffMins = Math.floor((new Date() - lastSync) / 60000);
    document.getElementById(
      "stat-last-sync"
    ).innerText = `Last sync: ${diffMins}m ago`;
  } catch (e) {
    console.warn("Sync error");
  }
}

async function handlePanicMode() {
  if (
    !confirm(
      "⚠️ CẢNH BÁO: Bạn có chắc chắn muốn kích hoạt PANIC MODE?\n\nHệ thống sẽ NGẮT toàn bộ AI Bot để tránh spam. Chỉ có chat thủ công hoạt động."
    )
  )
    return;

  try {
    const res = await fetch(`${API_BASE}/system/panic`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "enable", reason: "Admin Trigger" }),
    });
    if (res.ok) alert("🚨 ĐÃ KÍCH HOẠT PANIC MODE!");
    else alert("Lỗi kích hoạt Panic Mode");
  } catch (e) {
    alert("Mất kết nối Server");
  }
}

// --- UI NAVIGATION & LISTENERS ---

function setupEventListeners() {
  // Sidebar Toggle
  document.getElementById("toggle-sidebar")?.addEventListener("click", () => {
    const sb = document.getElementById("sidebar");
    sb.classList.toggle("w-64");
    sb.classList.toggle("w-20");
    document
      .querySelectorAll(".sidebar-text")
      .forEach((el) => el.classList.toggle("hidden"));
  });

  // Panic Button
  document
    .getElementById("panic-btn")
    ?.addEventListener("click", handlePanicMode);

  // Chat Events
  document
    .getElementById("message-input")
    ?.addEventListener("keypress", (e) => {
      if (e.key === "Enter") sendMessage();
    });
  document.getElementById("send-btn")?.addEventListener("click", sendMessage);
}

function switchView(viewName) {
  document
    .querySelectorAll('[id^="view-"]')
    .forEach((el) => el.classList.add("hidden"));
  document.getElementById(`view-${viewName}`).classList.remove("hidden");

  document.querySelectorAll(".nav-item").forEach((el) => {
    el.classList.remove("bg-blue-50", "text-blue-600");
    el.classList.add("text-gray-600");
  });
  const btn = document.querySelector(
    `button[onclick="switchView('${viewName}')"]`
  );
  if (btn) {
    btn.classList.remove("text-gray-600");
    btn.classList.add("bg-blue-50", "text-blue-600");
  }

  if (viewName === "facebook") loadConversations();
}

// --- CHAT LOGIC (PHASE 3) ---

async function loadConversations() {
  const container = document.getElementById("conversation-list");
  if (!container) return;

  try {
    const res = await fetch(`${API_BASE}/conversations`);
    if (!res.headers.get("content-type")?.includes("application/json"))
      throw new Error("API Error");

    const result = await res.json();
    container.innerHTML = "";

    if (!result.data || result.data.length === 0) {
      container.innerHTML =
        '<div class="p-4 text-center text-xs text-gray-400">Trống</div>';
      return;
    }

    result.data.forEach((conv) => {
      const div = document.createElement("div");
      div.className =
        "p-3 mx-2 my-1 flex items-center cursor-pointer hover:bg-gray-100 rounded-lg transition conversation-item";
      div.onclick = () => selectConversation(conv.id, conv.customer_name);
      const initial = (conv.customer_name || "?").charAt(0).toUpperCase();

      div.innerHTML = `
                <div class="w-10 h-10 rounded-full bg-blue-100 text-blue-600 flex items-center justify-center font-bold mr-3 shrink-0">${initial}</div>
                <div class="flex-1 min-w-0">
                    <h3 class="text-sm font-bold text-gray-800 truncate">${
                      conv.customer_name
                    }</h3>
                    <p class="text-xs text-gray-500 truncate">${
                      conv.last_message_content || "..."
                    }</p>
                </div>
            `;
      container.appendChild(div);
    });
  } catch (e) {
    container.innerHTML = `<div class="p-4 text-center text-red-500 text-xs">Lỗi kết nối</div>`;
  }
}

async function selectConversation(id, name) {
  currentConversationId = id;
  document.getElementById("header-name").innerText = name;
  document.getElementById("header-avatar").innerText = name
    .charAt(0)
    .toUpperCase();

  document.getElementById("welcome-screen").classList.add("hidden");
  document.getElementById("chat-header").classList.remove("hidden");
  document.getElementById("chat-messages").classList.remove("hidden");
  document.getElementById("input-area").classList.remove("hidden");

  const chatBox = document.getElementById("chat-messages");
  chatBox.innerHTML =
    '<div class="text-center mt-4 text-xs text-gray-400">Đang tải...</div>';

  try {
    const res = await fetch(`${API_BASE}/conversations/${id}/messages`);
    const result = await res.json();
    if (result.code === 200) renderMessages(result.data);
  } catch (e) {
    chatBox.innerHTML =
      '<div class="text-center text-red-500 text-xs mt-4">Lỗi tải tin nhắn</div>';
  }
}

function renderMessages(msgs) {
  const chatBox = document.getElementById("chat-messages");
  chatBox.innerHTML = '<div class="h-2"></div>';
  msgs.forEach((msg) => {
    const isMe = msg.sender_type === "agent";
    const div = document.createElement("div");
    div.className = `flex ${isMe ? "justify-end" : "justify-start"} mb-3`;
    div.innerHTML = `<div class="max-w-[75%] px-4 py-2 rounded-2xl text-sm ${
      isMe ? "bg-blue-600 text-white" : "bg-white border text-gray-800"
    }">${msg.content}</div>`;
    chatBox.appendChild(div);
  });
  chatBox.scrollTop = chatBox.scrollHeight;
}

async function sendMessage() {
  const input = document.getElementById("message-input");
  const text = input.value.trim();
  if (!text || !currentConversationId) return;

  const chatBox = document.getElementById("chat-messages");
  const temp = document.createElement("div");
  temp.className = "flex justify-end mb-3 opacity-50";
  temp.innerHTML = `<div class="max-w-[75%] px-4 py-2 rounded-2xl bg-blue-600 text-white text-sm">${text}</div>`;
  chatBox.appendChild(temp);
  chatBox.scrollTop = chatBox.scrollHeight;
  input.value = "";

  try {
    await fetch(`${API_BASE}/messages/reply`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        conversation_id: parseInt(currentConversationId),
        text,
      }),
    });
    temp.classList.remove("opacity-50");
  } catch (e) {
    temp.innerHTML = '<span class="text-red-500 text-xs">Lỗi gửi</span>';
  }
}
