// PWA App JavaScript
let agents = [];
let projects = [];
let sessions = [];
let notifications = [];
let isLoading = false;
let notificationInterval = null;
let websocket = null;
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;

// DOM elements
const loading = document.getElementById('loading');
const error = document.getElementById('error');
const agentsSection = document.getElementById('agents');
const agentDetail = document.getElementById('agent-detail');
const projectsSection = document.getElementById('projects');
const projectDetail = document.getElementById('project-detail');
const sessionsSection = document.getElementById('sessions');
const sessionDetail = document.getElementById('session-detail');
const notificationsPanel = document.getElementById('notifications-panel');
const agentGrid = document.getElementById('agent-grid');
const projectGrid = document.getElementById('project-grid');
const sessionsList = document.getElementById('sessions-list');
const agentCount = document.getElementById('agent-count');
const projectCount = document.getElementById('project-count');
const sessionCount = document.getElementById('session-count');
const errorMessage = document.getElementById('error-message');
const notificationBadge = document.getElementById('notification-badge');
const notificationsList = document.getElementById('notifications-list');
const connectionStatus = document.getElementById('status-indicator');

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
    updateConnectionStatus('connecting');
    loadAgents();
    loadNotifications();
    initWebSocket();
});

// Load agents from API
async function loadAgents() {
    if (isLoading) return;
    
    isLoading = true;
    showLoading();
    
    try {
        const response = await fetch('/api/agents');
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const data = await response.json();
        agents = data.agents || [];
        
        if (agents.length === 0) {
            showError('No agents found');
            return;
        }
        
        showAgents(agents);
        
    } catch (err) {
        console.error('Failed to load agents:', err);
        showError(`Failed to load agents: ${err.message}`);
    } finally {
        isLoading = false;
    }
}

// Show loading state
function showLoading() {
    hideAllSections();
    loading.style.display = 'block';
}

// Show error state
function showError(message) {
    hideAllSections();
    errorMessage.textContent = message;
    error.style.display = 'block';
}

// Show agents list
function showAgents(agentList) {
    hideAllSections();
    agentCount.textContent = `${agentList.length} agents`;
    
    // Clear existing grid
    agentGrid.innerHTML = '';
    
    // Create agent cards
    agentList.forEach(agent => {
        const card = createAgentCard(agent);
        agentGrid.appendChild(card);
    });
    
    agentsSection.style.display = 'block';
}

// Create agent card element
function createAgentCard(agent) {
    const card = document.createElement('div');
    card.className = 'agent-card';
    card.onclick = () => showAgentDetail(agent);
    
    const name = document.createElement('div');
    name.className = 'agent-name';
    name.textContent = agent.name;
    
    const type = document.createElement('div');
    type.className = 'agent-type';
    type.textContent = agent.type;
    
    const description = document.createElement('div');
    description.className = 'agent-description';
    description.textContent = agent.description;
    
    card.appendChild(name);
    card.appendChild(type);
    card.appendChild(description);
    
    return card;
}

// Show agent detail
async function showAgentDetail(agent) {
    try {
        // Load detailed agent information
        const response = await fetch(`/api/agents/${agent.type}`);
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const detailData = await response.json();
        
        // Update detail view
        document.getElementById('detail-name').textContent = detailData.name;
        document.getElementById('detail-description').textContent = detailData.description;
        document.getElementById('detail-system-prompt').textContent = detailData.system_prompt;
        document.getElementById('detail-task-prompt').textContent = detailData.task_prompt;
        
        hideAllSections();
        agentDetail.style.display = 'block';
        
    } catch (err) {
        console.error('Failed to load agent details:', err);
        showError(`Failed to load agent details: ${err.message}`);
    }
}

// Back to agents list
function backToList() {
    showAgents(agents);
}

// Load sessions from API
async function loadSessions() {
    if (isLoading) return;
    
    isLoading = true;
    showLoading();
    
    try {
        const response = await fetch('/api/sessions');
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const data = await response.json();
        sessions = data.sessions || [];
        
        if (sessions.length === 0) {
            showError('No sessions found');
            return;
        }
        
        displaySessions(sessions);
        
    } catch (err) {
        console.error('Failed to load sessions:', err);
        showError(`Failed to load sessions: ${err.message}`);
    } finally {
        isLoading = false;
    }
}

// Display sessions list
function displaySessions(sessionList) {
    hideAllSections();
    sessionCount.textContent = `${sessionList.length} sessions`;
    
    // Clear existing list
    sessionsList.innerHTML = '';
    
    // Create session items
    sessionList.forEach(session => {
        const item = createGlobalSessionItem(session);
        sessionsList.appendChild(item);
    });
    
    sessionsSection.style.display = 'block';
    updateActiveNav('sessions');
}

// Create session item element for global sessions view
function createGlobalSessionItem(session) {
    const item = document.createElement('div');
    item.className = 'session-item global-session clickable';
    item.onclick = () => showSessionDetail(session.id);
    
    const header = document.createElement('div');
    header.className = 'session-header';
    header.innerHTML = `
        <span class="session-agent">${session.agent_type}</span>
        <span class="session-status ${session.status}">${session.status}</span>
    `;
    
    const projectPath = document.createElement('div');
    projectPath.className = 'session-project';
    projectPath.textContent = session.project_path;
    
    const times = document.createElement('div');
    times.className = 'session-times';
    times.innerHTML = `
        <div>Started: ${formatDate(session.start_time)}</div>
        ${session.end_time ? `<div>Ended: ${formatDate(session.end_time)}</div>` : '<div>Active</div>'}
    `;
    
    item.appendChild(header);
    item.appendChild(projectPath);
    item.appendChild(times);
    
    return item;
}

// Load notifications from API
async function loadNotifications() {
    try {
        const response = await fetch('/api/notifications');
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const data = await response.json();
        notifications = data.notifications || [];
        
        updateNotificationBadge();
        renderNotifications();
        
    } catch (err) {
        console.error('Failed to load notifications:', err);
    }
}

// Update notification badge
function updateNotificationBadge() {
    const unreadCount = notifications.length;
    notificationBadge.textContent = unreadCount;
    notificationBadge.style.display = unreadCount > 0 ? 'block' : 'none';
}

// Render notifications in panel
function renderNotifications() {
    notificationsList.innerHTML = '';
    
    if (notifications.length === 0) {
        notificationsList.innerHTML = '<p style="text-align: center; color: #64748b; padding: 2rem;">No notifications</p>';
        return;
    }
    
    notifications.slice().reverse().forEach(notification => {
        const item = createNotificationItem(notification);
        notificationsList.appendChild(item);
    });
}

// Create notification item element
function createNotificationItem(notification) {
    const item = document.createElement('div');
    item.className = `notification-item ${notification.type}`;
    
    const header = document.createElement('div');
    header.className = 'notification-header';
    
    const title = document.createElement('div');
    title.className = 'notification-title';
    title.textContent = notification.title;
    
    const agent = document.createElement('div');
    agent.className = 'notification-agent';
    if (notification.agent) {
        agent.textContent = notification.agent;
    } else {
        agent.style.display = 'none';
    }
    
    header.appendChild(title);
    header.appendChild(agent);
    
    const message = document.createElement('div');
    message.className = 'notification-message';
    message.textContent = notification.message;
    
    const status = document.createElement('div');
    status.className = 'notification-status';
    if (notification.status) {
        status.textContent = `Status: ${notification.status}`;
    }
    
    item.appendChild(header);
    item.appendChild(message);
    if (notification.status) {
        item.appendChild(status);
    }
    
    return item;
}

// Toggle notifications panel
function toggleNotifications() {
    const isVisible = notificationsPanel.style.display === 'block';
    
    if (isVisible) {
        notificationsPanel.style.display = 'none';
    } else {
        notificationsPanel.style.display = 'block';
        loadNotifications(); // Refresh when opening
    }
}

// Initialize WebSocket connection
function initWebSocket() {
    if (!isOnline()) {
        console.log('App is offline, skipping WebSocket connection');
        setTimeout(initWebSocket, 5000);
        return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;

    try {
        websocket = new WebSocket(wsUrl);

        websocket.onopen = () => {
            console.log('WebSocket connection established');
            reconnectAttempts = 0;
            updateConnectionStatus('connected');
        };

        websocket.onmessage = (event) => {
            try {
                const notification = JSON.parse(event.data);
                handleRealTimeNotification(notification);
            } catch (err) {
                console.error('Failed to parse WebSocket message:', err);
            }
        };

        websocket.onclose = (event) => {
            console.log('WebSocket connection closed:', event.code, event.reason);
            websocket = null;
            updateConnectionStatus('disconnected');
            attemptReconnect();
        };

        websocket.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

    } catch (err) {
        console.error('Failed to create WebSocket connection:', err);
        fallbackToPolling();
    }
}

// Handle real-time notification from WebSocket
function handleRealTimeNotification(notification) {
    // Add notification to the beginning of the array
    notifications.unshift(notification);
    
    // Keep only last 100 notifications in memory
    if (notifications.length > 100) {
        notifications = notifications.slice(0, 100);
    }
    
    updateNotificationBadge();
    renderNotifications();
    
    // Show toast notification for immediate feedback
    showToastNotification(notification);
}

// Show toast notification
function showToastNotification(notification) {
    // Create toast element
    const toast = document.createElement('div');
    toast.className = 'toast-notification';
    toast.innerHTML = `
        <div class="toast-content">
            <div class="toast-title">${notification.title}</div>
            <div class="toast-message">${notification.message}</div>
        </div>
        <button class="toast-close" onclick="this.parentElement.remove()">×</button>
    `;
    
    // Add to document
    document.body.appendChild(toast);
    
    // Auto remove after 5 seconds
    setTimeout(() => {
        if (toast.parentElement) {
            toast.remove();
        }
    }, 5000);
}

// Attempt to reconnect WebSocket
function attemptReconnect() {
    if (reconnectAttempts >= maxReconnectAttempts) {
        console.log('Max reconnection attempts reached, falling back to polling');
        fallbackToPolling();
        return;
    }

    reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
    
    console.log(`Attempting to reconnect WebSocket in ${delay}ms (attempt ${reconnectAttempts}/${maxReconnectAttempts})`);
    
    setTimeout(initWebSocket, delay);
}

// Fallback to polling if WebSocket fails
function fallbackToPolling() {
    console.log('Falling back to notification polling');
    updateConnectionStatus('polling');
    startNotificationPolling();
}

// Update connection status indicator
function updateConnectionStatus(status) {
    if (!connectionStatus) return;
    
    connectionStatus.className = 'status-indicator';
    
    switch (status) {
        case 'connected':
            connectionStatus.classList.add('connected');
            connectionStatus.title = 'Connected via WebSocket - Real-time notifications active';
            break;
        case 'connecting':
            connectionStatus.classList.add('connecting');
            connectionStatus.title = 'Connecting to WebSocket...';
            break;
        case 'disconnected':
            connectionStatus.classList.add('disconnected');
            connectionStatus.title = 'WebSocket disconnected - Attempting to reconnect';
            break;
        case 'polling':
            connectionStatus.classList.add('polling');
            connectionStatus.title = 'Polling mode - WebSocket unavailable';
            break;
        default:
            connectionStatus.title = 'Connection status unknown';
    }
}

// Initialize connection status
updateConnectionStatus('connecting');

// Start notification polling (fallback)
function startNotificationPolling() {
    // Clear any existing interval
    if (notificationInterval) {
        clearInterval(notificationInterval);
    }
    
    // Poll for new notifications every 30 seconds
    notificationInterval = setInterval(() => {
        if (document.hidden || !isOnline()) return;
        loadNotifications();
    }, 30000);
}

// Show projects view
async function showProjectsView() {
    if (isLoading) return;
    
    isLoading = true;
    showLoading();
    
    try {
        const response = await fetch('/api/projects');
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const data = await response.json();
        projects = data.projects || [];
        
        if (projects.length === 0) {
            showError('No projects found');
            return;
        }
        
        displayProjects(projects);
        
    } catch (err) {
        console.error('Failed to load projects:', err);
        showError(`Failed to load projects: ${err.message}`);
    } finally {
        isLoading = false;
    }
}

// Display projects list
function displayProjects(projectList) {
    hideAllSections();
    projectCount.textContent = `${projectList.length} projects`;
    
    // Clear existing grid
    projectGrid.innerHTML = '';
    
    // Create project cards
    projectList.forEach(project => {
        const card = createProjectCard(project);
        projectGrid.appendChild(card);
    });
    
    projectsSection.style.display = 'block';
    updateActiveNav('projects');
}

// Create project card element
function createProjectCard(project) {
    const card = document.createElement('div');
    card.className = 'project-card';
    card.onclick = () => showProjectDetail(project);
    
    const path = document.createElement('div');
    path.className = 'project-path';
    path.textContent = project.project_path;
    
    const stats = document.createElement('div');
    stats.className = 'project-stats-card';
    stats.innerHTML = `
        <div class="stat-item">
            <span class="stat-label">Sessions:</span>
            <span class="stat-value">${project.total_sessions}</span>
        </div>
        <div class="stat-item">
            <span class="stat-label">Last seen:</span>
            <span class="stat-value">${formatDate(project.last_seen)}</span>
        </div>
    `;
    
    card.appendChild(path);
    card.appendChild(stats);
    
    return card;
}

// Show project detail
async function showProjectDetail(project) {
    try {
        // Load detailed project information
        const encodedPath = encodeURIComponent(project.project_path);
        const response = await fetch(`/api/projects/${encodedPath}`);
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const detailData = await response.json();
        
        // Update detail view
        const projectInfo = detailData.project;
        document.getElementById('project-detail-path').textContent = projectInfo.project_path;
        
        // Display project statistics
        const statsContainer = document.getElementById('project-stats');
        statsContainer.innerHTML = `
            <div class="stats-grid">
                <div class="stat-box">
                    <h4>Total Sessions</h4>
                    <p>${projectInfo.total_sessions}</p>
                </div>
                <div class="stat-box">
                    <h4>First Seen</h4>
                    <p>${formatDate(projectInfo.first_seen)}</p>
                </div>
                <div class="stat-box">
                    <h4>Last Seen</h4>
                    <p>${formatDate(projectInfo.last_seen)}</p>
                </div>
            </div>
        `;
        
        // Load and display project tasks
        await loadProjectTasks(project.project_path);
        
        // Display recent sessions
        const sessionsContainer = document.getElementById('project-sessions-list');
        if (projectInfo.recent_sessions && projectInfo.recent_sessions.length > 0) {
            sessionsContainer.innerHTML = '';
            projectInfo.recent_sessions.forEach(session => {
                const sessionItem = createSessionItem(session);
                sessionsContainer.appendChild(sessionItem);
            });
        } else {
            sessionsContainer.innerHTML = '<p style="text-align: center; color: #64748b; padding: 1rem;">No recent sessions</p>';
        }
        
        hideAllSections();
        projectDetail.style.display = 'block';
        
    } catch (err) {
        console.error('Failed to load project details:', err);
        showError(`Failed to load project details: ${err.message}`);
    }
}

// Create session item element
function createSessionItem(session) {
    const item = document.createElement('div');
    item.className = 'session-item clickable';
    item.onclick = () => showSessionDetail(session.id);
    
    const header = document.createElement('div');
    header.className = 'session-header';
    header.innerHTML = `
        <span class="session-agent">${session.agent_type}</span>
        <span class="session-status ${session.status}">${session.status}</span>
    `;
    
    const times = document.createElement('div');
    times.className = 'session-times';
    times.innerHTML = `
        <div>Started: ${formatDate(session.start_time)}</div>
        ${session.end_time ? `<div>Ended: ${formatDate(session.end_time)}</div>` : '<div>Active</div>'}
    `;
    
    item.appendChild(header);
    item.appendChild(times);
    
    return item;
}

// Show session detail
async function showSessionDetail(sessionId) {
    try {
        const response = await fetch(`/api/sessions/${sessionId}`);
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const data = await response.json();
        const session = data.session;
        
        // Update detail view
        document.getElementById('session-detail-agent').textContent = session.agent_type;
        document.getElementById('session-detail-status').textContent = session.status;
        document.getElementById('session-detail-status').className = `session-status ${session.status}`;
        document.getElementById('session-detail-project').textContent = session.project_path;
        document.getElementById('session-detail-start-time').textContent = formatDate(session.start_time);
        document.getElementById('session-detail-end-time').textContent = session.end_time ? formatDate(session.end_time) : 'Still active';
        
        // Display conversation
        const conversationContainer = document.getElementById('session-conversation');
        conversationContainer.innerHTML = '';
        
        if (session.inputs && session.inputs.length > 0 && session.outputs && session.outputs.length > 0) {
            // Interleave inputs and outputs chronologically
            const maxItems = Math.max(session.inputs.length, session.outputs.length);
            for (let i = 0; i < maxItems; i++) {
                if (i < session.inputs.length) {
                    const inputDiv = document.createElement('div');
                    inputDiv.className = 'conversation-message input';
                    inputDiv.innerHTML = `
                        <div class="message-label">Input:</div>
                        <div class="message-content">${escapeHtml(session.inputs[i])}</div>
                    `;
                    conversationContainer.appendChild(inputDiv);
                }
                
                if (i < session.outputs.length) {
                    const outputDiv = document.createElement('div');
                    outputDiv.className = 'conversation-message output';
                    outputDiv.innerHTML = `
                        <div class="message-label">Output:</div>
                        <div class="message-content">${escapeHtml(session.outputs[i])}</div>
                    `;
                    conversationContainer.appendChild(outputDiv);
                }
            }
        } else {
            conversationContainer.innerHTML = '<p style="text-align: center; color: #64748b; padding: 2rem;">No conversation data available</p>';
        }
        
        hideAllSections();
        sessionDetail.style.display = 'block';
        
    } catch (err) {
        console.error('Failed to load session details:', err);
        showError(`Failed to load session details: ${err.message}`);
    }
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Format date helper function
function formatDate(dateString) {
    if (!dateString) return 'Unknown';
    const date = new Date(dateString);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
}

// Navigation functions
function showAgents() {
    showAgentsList(agents);
}

function showProjects() {
    showProjectsView();
}

function showSessions() {
    loadSessions();
}

function backToProjects() {
    showProjectsView();
}

function backToSessions() {
    loadSessions();
}

// Update active navigation state
function updateActiveNav(activeSection) {
    const agentsBtn = document.getElementById('agents-nav-btn');
    const projectsBtn = document.getElementById('projects-nav-btn');
    const sessionsBtn = document.getElementById('sessions-nav-btn');
    
    // Remove active class from all buttons
    agentsBtn.classList.remove('active');
    projectsBtn.classList.remove('active');
    sessionsBtn.classList.remove('active');
    
    // Add active class to current button
    if (activeSection === 'agents') {
        agentsBtn.classList.add('active');
    } else if (activeSection === 'projects') {
        projectsBtn.classList.add('active');
    } else if (activeSection === 'sessions') {
        sessionsBtn.classList.add('active');
    }
}

// Hide all sections
function hideAllSections() {
    loading.style.display = 'none';
    error.style.display = 'none';
    agentsSection.style.display = 'none';
    agentDetail.style.display = 'none';
    projectsSection.style.display = 'none';
    projectDetail.style.display = 'none';
    sessionsSection.style.display = 'none';
    sessionDetail.style.display = 'none';
    // Don't hide notifications panel here as it should be toggleable
}

// Online/offline detection
window.addEventListener('online', () => {
    console.log('App is online');
    // Could add a toast notification here
});

window.addEventListener('offline', () => {
    console.log('App is offline');
    // Could add a toast notification here
});

// Check connection status
function isOnline() {
    return navigator.onLine;
}

// Pull-to-refresh functionality
let startY = 0;
let isPulling = false;

document.addEventListener('touchstart', (e) => {
    if (window.scrollY === 0) {
        startY = e.touches[0].pageY;
        isPulling = true;
    }
});

document.addEventListener('touchmove', (e) => {
    if (!isPulling) return;
    
    const currentY = e.touches[0].pageY;
    const diff = currentY - startY;
    
    if (diff > 0 && diff < 150) {
        document.body.style.transform = `translateY(${diff * 0.5}px)`;
    }
});

document.addEventListener('touchend', () => {
    if (!isPulling) return;
    
    isPulling = false;
    document.body.style.transform = '';
    
    // If pulled down enough, refresh
    const diff = e.changedTouches[0].pageY - startY;
    if (diff > 100) {
        loadAgents();
    }
});

// Load project tasks
async function loadProjectTasks(projectPath) {
    try {
        const encodedPath = encodeURIComponent(projectPath);
        const response = await fetch(`/api/projects/${encodedPath}/tasks`);
        
        if (!response.ok) {
            // If tasks file doesn't exist, show a message
            if (response.status === 404) {
                const tasksContainer = document.getElementById('project-tasks-content');
                tasksContainer.innerHTML = `
                    <div class="no-tasks">
                        <p>No tasks.md file found in this project.</p>
                        <p>Click "Add Task" to create one and add your first task.</p>
                    </div>
                `;
                return;
            }
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const data = await response.json();
        
        // Display tasks content
        const tasksContainer = document.getElementById('project-tasks-content');
        if (data.tasks && data.tasks.trim() !== '') {
            tasksContainer.innerHTML = `
                <div class="tasks-content">
                    <pre class="tasks-text">${escapeHtml(data.tasks)}</pre>
                </div>
            `;
        } else {
            tasksContainer.innerHTML = `
                <div class="no-tasks">
                    <p>No tasks found in this project.</p>
                    <p>Click "Add Task" to add your first task.</p>
                </div>
            `;
        }
        
    } catch (err) {
        console.error('Failed to load project tasks:', err);
        const tasksContainer = document.getElementById('project-tasks-content');
        tasksContainer.innerHTML = `
            <div class="error-message">
                <p>Failed to load tasks: ${err.message}</p>
            </div>
        `;
    }
}

// Show add task form
function showAddTaskForm() {
    const modal = document.getElementById('add-task-modal');
    modal.style.display = 'block';
    document.getElementById('task-input').focus();
}

// Hide add task form
function hideAddTaskForm() {
    const modal = document.getElementById('add-task-modal');
    modal.style.display = 'none';
    document.getElementById('add-task-form').reset();
}

// Add task to project
async function addTask(event) {
    event.preventDefault();
    
    const formData = new FormData(event.target);
    const task = formData.get('task').trim();
    const priority = formData.get('priority');
    
    if (!task) {
        alert('Task description is required');
        return;
    }
    
    // Get current project path from the detail view
    const projectPath = document.getElementById('project-detail-path').textContent;
    
    try {
        const encodedPath = encodeURIComponent(projectPath);
        const response = await fetch(`/api/projects/${encodedPath}/tasks`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                task: task,
                priority: priority
            })
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const result = await response.json();
        
        // Show success message
        showToastNotification({
            type: 'success',
            title: 'Task Added',
            message: `Successfully added task to project: ${projectPath}`
        });
        
        // Hide the form
        hideAddTaskForm();
        
        // Reload tasks to show the new task
        await loadProjectTasks(projectPath);
        
    } catch (err) {
        console.error('Failed to add task:', err);
        alert(`Failed to add task: ${err.message}`);
    }
}

// Handle visibility changes (app switching)
document.addEventListener('visibilitychange', () => {
    if (!document.hidden && isOnline()) {
        // Refresh data when app becomes visible again
        loadAgents();
    }
});