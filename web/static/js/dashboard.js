// Immortal Chat OS Dashboard - Main JavaScript
// Mobile-first, real-time dashboard with API integration

class Dashboard {
  constructor() {
    this.apiBase = "/api";
    this.refreshInterval = 5000; // 5 seconds
    this.timers = [];
    this.init();
  }

  init() {
    console.log("🧬 Immortal Chat OS Dashboard initializing...");

    // Initial data load
    this.loadAllData();

    // Setup auto-refresh
    this.startAutoRefresh();

    // Setup event listeners
    this.setupEventListeners();

    console.log("✅ Dashboard ready");
  }

  async loadAllData() {
    try {
      await Promise.all([
        this.loadSystemStatus(),
        this.loadSystemMetrics(),
        this.loadPlatforms(),
        this.loadSyncStatus(),
      ]);
    } catch (error) {
      console.error("Failed to load dashboard data:", error);
      this.showError("Failed to connect to server");
    }
  }

  async fetchAPI(endpoint) {
    const response = await fetch(`${this.apiBase}/${endpoint}`);
    if (!response.ok) {
      throw new Error(`API error: ${response.statusText}`);
    }
    return response.json();
  }

  async loadSystemStatus() {
    try {
      const data = await this.fetchAPI("status");

      document.getElementById("systemStatus").textContent = data.online
        ? "Online"
        : "Offline";
      document.getElementById("systemUptime").textContent = data.uptime;
      document.getElementById("systemVersion").textContent = data.version;
      document.getElementById("tenantId").textContent = data.tenant_id;
    } catch (error) {
      console.error("Failed to load system status:", error);
    }
  }

  async loadSystemMetrics() {
    try {
      const data = await this.fetchAPI("system/metrics");

      // CPU
      this.updateProgress(
        "cpuProgress",
        data.cpu_percent,
        "cpuValue",
        `${data.cpu_percent.toFixed(1)}%`
      );

      // RAM
      const ramPercent = (data.ram_used_gb / data.ram_total_gb) * 100;
      this.updateProgress(
        "ramProgress",
        ramPercent,
        "ramValue",
        `${data.ram_used_gb.toFixed(1)} / ${data.ram_total_gb.toFixed(1)} GB`
      );

      // Disk (with color coding)
      const diskProgress = document.getElementById("diskProgress");
      this.updateProgress(
        "diskProgress",
        data.disk_percent,
        "diskValue",
        `${data.disk_percent.toFixed(1)}% (${data.disk_used_gb.toFixed(1)} GB)`
      );

      // Color code disk based on usage
      diskProgress.classList.remove("warning", "critical");
      if (data.disk_warning_level === "warning") {
        diskProgress.classList.add("warning");
      } else if (data.disk_warning_level === "critical") {
        diskProgress.classList.add("critical");
      }

      // Goroutines
      document.getElementById("goroutines").textContent = data.goroutines_count;
    } catch (error) {
      console.error("Failed to load system metrics:", error);
    }
  }

  async loadPlatforms() {
    try {
      const platforms = await this.fetchAPI("platforms");

      // Render platform list
      const platformList = document.getElementById("platformList");
      platformList.innerHTML = platforms
        .map(
          (p) => `
                <div class="platform-item" data-platform-id="${p.id}">
                    <span class="status-dot ${this.getStatusClass(
                      p.status
                    )}"></span>
                    <span class="platform-icon">${this.getPlatformIcon(
                      p.platform
                    )}</span>
                    <span class="platform-name">${p.name}</span>
                </div>
            `
        )
        .join("");

      // Load first platform details
      if (platforms.length > 0) {
        this.loadPlatformDetails(platforms[0]);
      }

      // Add click handlers
      platformList.querySelectorAll(".platform-item").forEach((item) => {
        item.addEventListener("click", () => {
          const platform = platforms.find(
            (p) => p.id == item.dataset.platformId
          );
          if (platform) this.loadPlatformDetails(platform);
        });
      });
    } catch (error) {
      console.error("Failed to load platforms:", error);
    }
  }

  loadPlatformDetails(platform) {
    document.getElementById("detailIcon").textContent = this.getPlatformIcon(
      platform.platform
    );
    document.getElementById("detailName").textContent = platform.name;
    document.getElementById("detailStatusText").textContent =
      this.capitalizeFirst(platform.status);

    const statusDot = document.getElementById("detailStatus");
    statusDot.className = `status-dot ${this.getStatusClass(platform.status)}`;

    // Statistics
    document.getElementById("messagesToday").textContent =
      platform.message_count_today || "--";
    document.getElementById("messagesTotal").textContent = "1,450"; // TODO: From API
    document.getElementById("pendingSync").textContent =
      platform.pending_sync || "0";
    document.getElementById("successRate").textContent = "98%"; // TODO: From API
  }

  async loadSyncStatus() {
    try {
      const data = await this.fetchAPI("sync/status");

      document.getElementById("syncPending").textContent =
        data.pending_messages;

      const syncHealth = document.getElementById("syncHealth");
      syncHealth.className = "status-dot";

      if (data.sync_health === "healthy") {
        syncHealth.classList.add("healthy");
      } else if (data.sync_health === "lagging") {
        syncHealth.classList.add("lagging");
      } else {
        syncHealth.classList.add("critical");
      }

      // Format last sync time
      const lastSync = new Date(data.last_sync_at);
      const now = new Date();
      const diffMinutes = Math.floor((now - lastSync) / 1000 / 60);
      document.getElementById("syncTime").textContent = `${diffMinutes}m ago`;
    } catch (error) {
      console.error("Failed to load sync status:", error);
    }
  }

  updateProgress(progressId, percent, valueId, text) {
    const progress = document.getElementById(progressId);
    const value = document.getElementById(valueId);

    if (progress) {
      progress.style.width = `${Math.min(percent, 100)}%`;
    }

    if (value) {
      value.textContent = text;
    }
  }

  startAutoRefresh() {
    // Refresh metrics more frequently
    const metricsTimer = setInterval(
      () => this.loadSystemMetrics(),
      this.refreshInterval
    );
    const syncTimer = setInterval(
      () => this.loadSyncStatus(),
      this.refreshInterval
    );
    const platformsTimer = setInterval(
      () => this.loadPlatforms(),
      this.refreshInterval * 2
    ); // Less frequent

    this.timers.push(metricsTimer, syncTimer, platformsTimer);
  }

  setupEventListeners() {
    // Panic button
    const panicBtn = document.getElementById("panicBtn");
    panicBtn.addEventListener("click", () => this.handlePanicMode());

    // Mobile bottom navigation
    document.querySelectorAll(".nav-btn").forEach((btn) => {
      btn.addEventListener("click", () => {
        // Remove active from all
        document
          .querySelectorAll(".nav-btn")
          .forEach((b) => b.classList.remove("active"));
        // Add active to clicked
        btn.classList.add("active");

        // Handle view switching (for mobile)
        const view = btn.dataset.view;
        this.switchMobileView(view);
      });
    });
  }

  switchMobileView(view) {
    // Simple show/hide for mobile views
    // In production, you'd want more sophisticated view management
    console.log("Switching to view:", view);
  }

  async handlePanicMode() {
    const confirmed = confirm(
      "⚠️ WARNING\n\nThis will disable all AI modules!\nOnly manual messaging will work.\n\nAre you sure?"
    );

    if (!confirmed) return;

    try {
      const response = await fetch(`${this.apiBase}/system/panic`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          action: "enable",
          reason: "Manual trigger from dashboard",
        }),
      });

      if (response.ok) {
        alert(
          "🚨 PANIC MODE ACTIVATED\n\nAI modules disabled.\nManual mode only."
        );
      } else {
        alert("Failed to activate panic mode");
      }
    } catch (error) {
      console.error("Panic mode error:", error);
      alert("Error: Could not activate panic mode");
    }
  }

  getStatusClass(status) {
    const map = {
      connected: "healthy",
      warning: "lagging",
      error: "critical",
      offline: "offline",
    };
    return map[status] || "offline";
  }

  getPlatformIcon(platform) {
    const icons = {
      facebook: "📘",
      zalo: "💬",
      telegram: "📱",
    };
    return icons[platform] || "📱";
  }

  capitalizeFirst(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
  }

  showError(message) {
    console.error("Dashboard error:", message);
    // TODO: Show toast notification
  }

  destroy() {
    this.timers.forEach((timer) => clearInterval(timer));
    console.log("Dashboard destroyed");
  }
}

// Initialize dashboard when DOM is ready
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", () => {
    window.dashboard = new Dashboard();
  });
} else {
  window.dashboard = new Dashboard();
}
