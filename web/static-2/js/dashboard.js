// --- CẤU HÌNH ---
const API_BASE = "/api";
let currentConversationId = null;
let currentView = "dashboard";

// --- KHỞI ĐỘNG ---
document.addEventListener("DOMContentLoaded", () => {
  // 1. Mặc định vào Dashboard
  switchView("dashboard");

  // 2. Chạy giả lập Health Check (Nếu chưa có API thật)
  startSystemHealthMonitor();

  // 3. Sự kiện Toggle Sidebar
  const toggleBtn = document.getElementById("toggle-sidebar");
  const sidebar = document.getElementById("sidebar");
  if (toggleBtn && sidebar) {
    toggleBtn.addEventListener("click", () => {
      sidebar.classList.toggle("w-64");
      sidebar.classList.toggle("w-20");
      // Ẩn hiện text khi thu nhỏ
      document
        .querySelectorAll(".sidebar-text")
        .forEach((el) => el.classList.toggle("hidden"));
    });
  }

  // 4. Sự kiện Chat Input
  const textarea = document.getElementById("message-input");
  if (textarea) {
    textarea.addEventListener("input", function () {
      this.style.height = "auto";
      this.style.height = this.scrollHeight + "px";
    });
    textarea.addEventListener("keydown", (e) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
      }
    });
  }
  document.getElementById("send-btn")?.addEventListener("click", sendMessage);
});

// --- NAVIGATION LOGIC ---
function switchView(viewName) {
  currentView = viewName;

  // 1. Update UI Sidebar
  document.querySelectorAll(".nav-item").forEach((el) => {
    el.classList.remove("bg-blue-50", "text-blue-600", "active-nav");
    el.classList.add("text-gray-600");
  });

  // Highlight active tab
  // Tìm button gọi hàm này để highlight (đơn giản hóa)
  const activeBtn = document.querySelector(
    `button[onclick="switchView('${viewName}')"]`
  );
  if (activeBtn) {
    activeBtn.classList.add("bg-blue-50", "text-blue-600", "active-nav");
    activeBtn.classList.remove("text-gray-600");
  }

  // 2. Show/Hide Views
  document.getElementById("view-dashboard").classList.add("hidden");
  document.getElementById("view-facebook").classList.add("hidden");

  document.getElementById(`view-${viewName}`).classList.remove("hidden");

  // 3. Load Data nếu cần
  if (viewName === "facebook") {
    loadConversations();
  }
}

// --- SYSTEM HEALTH MONITOR (MOCKUP OR REAL) ---
function startSystemHealthMonitor() {
  // Hàm cập nhật UI
  const updateHealth = (cpu, ram) => {
    const cpuEl = document.getElementById("metric-cpu");
    const ramEl = document.getElementById("metric-ram");
    const cpuBar = document.getElementById("bar-cpu");
    const ramBar = document.getElementById("bar-ram");

    if (cpuEl) cpuEl.innerText = `${cpu}%`;
    if (ramEl) ramEl.innerText = `${ram}%`;
    if (cpuBar) cpuBar.style.width = `${cpu}%`;
    if (ramBar) ramBar.style.width = `${ram}%`;

    // Đổi màu nếu cao
    if (cpu > 80 && cpuBar)
      cpuBar.classList.replace("bg-blue-500", "bg-red-500");
    else if (cpuBar) cpuBar.classList.replace("bg-red-500", "bg-blue-500");
  };

  // Gọi API thật nếu có, không thì random để demo sức sống
  setInterval(async () => {
    try {
      // Thử gọi API thật (Nếu bạn đã làm ở Phase 2)
      // const res = await fetch('/api/system/metrics');
      // const data = await res.json();
      // updateHealth(data.cpu, data.ram);

      // DEMO: Random số liệu để nhìn cho "động"
      const mockCpu = Math.floor(Math.random() * 30) + 10; // 10-40%
      const mockRam = Math.floor(Math.random() * 20) + 40; // 40-60%
      updateHealth(mockCpu, mockRam);
    } catch (e) {
      console.log("Health monitor error");
    }
  }, 3000);
}

// --- CHAT LOGIC (TỪ PHASE 3) ---

async function loadConversations() {
  const listContainer = document.getElementById("conversation-list");
  if (!listContainer) return;

  try {
    const response = await fetch(`${API_BASE}/conversations`);
    if (!response.ok) throw new Error("API Error");
    const result = await response.json();

    if (result.code === 200) {
      if (result.data && result.data.length > 0) {
        renderList(result.data);
      } else {
        listContainer.innerHTML = `<div class="p-8 text-center text-gray-400 text-xs">Trống</div>`;
      }
    }
  } catch (error) {
    listContainer.innerHTML = `<div class="p-4 text-center text-red-500 text-xs">Mất kết nối</div>`;
  }
}

function renderList(conversations) {
  const listContainer = document.getElementById("conversation-list");
  listContainer.innerHTML = "";

  conversations.forEach((conv) => {
    const div = document.createElement("div");
    div.className = `p-3 mx-2 my-1 flex items-center cursor-pointer hover:bg-gray-100 rounded-lg transition-colors conversation-item group`;
    div.dataset.id = conv.id;
    div.onclick = () => selectConversation(conv.id, conv.customer_name);

    const initial = conv.customer_name
      ? conv.customer_name.charAt(0).toUpperCase()
      : "?";
    const isUnread = conv.status === "unread";

    div.innerHTML = `
            <div class="w-10 h-10 rounded-full ${
              isUnread ? "bg-blue-600 text-white" : "bg-gray-200 text-gray-600"
            } flex items-center justify-center text-sm font-bold mr-3 shrink-0">
                ${initial}
            </div>
            <div class="flex-1 min-w-0">
                <div class="flex justify-between items-baseline mb-0.5">
                    <h3 class="text-sm font-bold text-gray-800 truncate">${
                      conv.customer_name || "Khách hàng"
                    }</h3>
                    <span class="text-[10px] text-gray-400 shrink-0">vừa xong</span>
                </div>
                <p class="text-xs text-gray-500 truncate opacity-90">${
                  conv.last_message_content || "..."
                }</p>
            </div>
        `;
    listContainer.appendChild(div);
  });
}

async function selectConversation(id, name) {
  currentConversationId = id;

  // Highlight
  document
    .querySelectorAll(".conversation-item")
    .forEach((el) =>
      el.classList.remove("bg-blue-50", "ring-1", "ring-blue-200")
    );
  document
    .querySelector(`.conversation-item[data-id="${id}"]`)
    ?.classList.add("bg-blue-50", "ring-1", "ring-blue-200");

  // Show Chat UI
  document.getElementById("welcome-screen").classList.add("hidden");
  document.getElementById("chat-header").classList.remove("hidden");
  document.getElementById("chat-messages").classList.remove("hidden");
  document.getElementById("input-area").classList.remove("hidden");

  // Update Header
  document.getElementById("header-name").innerText = name || "Khách hàng";
  document.getElementById("header-avatar").innerText = (name || "?")
    .charAt(0)
    .toUpperCase();

  // Load Messages
  const chatBox = document.getElementById("chat-messages");
  chatBox.innerHTML =
    '<div class="text-center text-xs text-gray-400 mt-4">Đang tải tin nhắn...</div>';

  try {
    const res = await fetch(`${API_BASE}/conversations/${id}/messages`);
    const result = await res.json();
    if (result.code === 200) renderMessages(result.data);
  } catch (e) {
    chatBox.innerHTML =
      '<div class="text-center text-red-400 text-xs mt-4">Lỗi tải tin nhắn</div>';
  }
}

function renderMessages(messages) {
  const chatBox = document.getElementById("chat-messages");
  chatBox.innerHTML = '<div class="h-2"></div>';

  messages.forEach((msg) => {
    const isAgent = msg.sender_type === "agent";
    const div = document.createElement("div");
    div.className = `flex ${
      isAgent ? "justify-end" : "justify-start"
    } mb-3 px-2`;

    div.innerHTML = `
            <div class="max-w-[75%] px-4 py-2 rounded-2xl text-[14px] ${
              isAgent
                ? "bg-blue-600 text-white rounded-br-none"
                : "bg-white border border-gray-200 text-gray-800 rounded-bl-none shadow-sm"
            }">
                ${msg.content}
            </div>
        `;
    chatBox.appendChild(div);
  });
  chatBox.scrollTop = chatBox.scrollHeight;
}

async function sendMessage() {
  const input = document.getElementById("message-input");
  const text = input.value.trim();
  if (!text || !currentConversationId) return;

  // Optimistic UI
  const chatBox = document.getElementById("chat-messages");
  const tempDiv = document.createElement("div");
  tempDiv.className = "flex justify-end mb-3 px-2 opacity-50";
  tempDiv.innerHTML = `<div class="max-w-[75%] px-4 py-2 rounded-2xl bg-blue-600 text-white rounded-br-none text-[14px]">${text}</div>`;
  chatBox.appendChild(tempDiv);
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
    tempDiv.classList.remove("opacity-50"); // Thành công
  } catch (e) {
    tempDiv.innerHTML = '<span class="text-red-500 text-xs">Lỗi gửi tin</span>';
  }
}
