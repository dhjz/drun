const { createApp } = Vue

const STORAGE_KEY = 'project_manager_data'

const PRESET_COLORS = [
  '#409eff', '#67c23a', '#e6a23c', '#f56c6c', '#909399',
  '#00bcd4', '#9c27b0', '#ff5722', '#795548', '#607d8b',
  '#e91e63', '#3f51b5', '#009688', '#8bc34a', '#ffc107',
  '#ff9800', '#ffeb3b', '#cddc39', '#03a9f4', '#2196f3'
]

var app = createApp({
  data() {
    return {
      projects: [],
      tags: [],
      showModal: false,
      showLogPanel: false,
      showSubModulePanel: false,
      showAddTagModal: false,
      showAddCommandModal: false,
      selectedProject: null,
      editingProject: null,
      logContent: '',
      logFile: '',
      form: {
        name: '',
        path: '',
        type: 'backend',
        commands: [],
        logFile: '',
        packageName: '',
        preCommand: 'git pull',
        subModules: [],
        sdkPath: '',
        tagIds: []
      },
      newCommand: {
        name: '',
        cmd: '',
        workDir: '',
        wait: false
      },
      newSubModule: {
        name: '',
        path: '',
        jarPath: ''
      },
      newTag: {
        name: '',
        color: '#409eff'
      },
      commandForm: {
        name: '',
        path: '',
        cmd: '',
        tagIds: []
      },
      editingCommandProject: null,
      presetColors: PRESET_COLORS,
      editingCommandIndex: -1,
      runningProjects: {},
      analyzing: false,
      searchKeyword: '',
      simpleMode: localStorage.getItem('simpleMode') === 'true',
      filterType: '',
      filterTag: ''
    }
  },
  computed: {
    hasProjects() {
      return this.projects.length > 0
    },
    filteredProjects() {
      let result = this.projects
      if (this.filterType) {
        result = result.filter(p => p.type === this.filterType)
      }
      if (this.filterTag) {
        result = result.filter(p => p.tagIds && p.tagIds.includes(this.filterTag))
      }
      if (this.searchKeyword) {
        const keyword = this.searchKeyword.toLowerCase()
        result = result.filter(p =>
          p.name.toLowerCase().includes(keyword) ||
          p.path.toLowerCase().includes(keyword)
        )
      }
      return result
    }
  },
  mounted() {
    this.loadProjects()
    this.startStatusCheck()
  },
  methods: {
    loadProjects() {
      fetch('/api/loadProjects')
        .then(res => res.json())
        .then(data => {
          if (data) {
            this.projects = data.projects || []
            this.tags = data.tags || []
            localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
          }
        })
        .catch(() => {
          const localData = loadData(STORAGE_KEY)
          if (localData) {
            this.projects = localData.projects || localData || []
            this.tags = localData.tags || []
          }
        })
    },
    async saveProjects() {
      const data = {
        projects: this.projects,
        tags: this.tags
      }
      try {
        const response = await fetch('/api/saveProjects', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(data)
        })
        const result = await response.json()
        if (result.success) {
          localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
        }
      } catch (e) {
        saveData(STORAGE_KEY, data)
      }
    },
    openAddModal() {
      this.editingProject = null
      this.form = {
        name: '',
        path: '',
        type: 'backend',
        commands: [],
        logFile: '',
        packageName: '',
        preCommand: 'git pull',
        subModules: [],
        sdkPath: '',
        tagIds: []
      }
      this.newCommand = { name: '', cmd: '', workDir: '', wait: false }
      this.newSubModule = { name: '', path: '', jarPath: '' }
      this.editingCommandIndex = -1
      this.showModal = true
    },
    openEditModal(project) {
      if (project.type === 'command') {
        this.editingCommandProject = project
        this.commandForm = {
          name: project.name,
          path: project.path,
          cmd: project.commands && project.commands.length > 0 ? project.commands[0].cmd : '',
          tagIds: project.tagIds ? [...project.tagIds] : []
        }
        this.showAddCommandModal = true
        return
      }
      this.editingProject = project
      this.form = {
        name: project.name,
        path: project.path,
        type: project.type || 'backend',
        commands: project.commands ? [...project.commands] : [],
        logFile: project.logFile || '',
        packageName: project.packageName || '',
        preCommand: project.preCommand || 'git pull',
        subModules: project.subModules ? project.subModules.map(m => ({...m})) : [],
        sdkPath: project.sdkPath || '',
        tagIds: project.tagIds ? [...project.tagIds] : []
      }
      this.newCommand = { name: '', cmd: '', workDir: '', wait: false }
      this.newSubModule = { name: '', path: '', jarPath: '' }
      this.editingCommandIndex = -1
      this.showModal = true
    },
    closeModal() {
      this.showModal = false
      this.editingProject = null
    },
    async browsePath() {
      try {
        const result = await apiSelectDir()
        if (result.path) {
          this.form.path = result.path
          await this.analyzeProject()
        }
      } catch (e) {
        showToast('选择目录失败: ' + e.message, 'error')
      }
    },
    async browseSdkPath(type) {
      try {
        const result = await apiSelectDir()
        if (result.path) {
          this.form.sdkPath = result.path
        }
      } catch (e) {
        showToast('选择目录失败: ' + e.message, 'error')
      }
    },
    async analyzeProject() {
      if (!this.form.path) {
        showToast('请输入项目路径', 'warning')
        return
      }
      const exists = await apiExists(this.form.path)
      if (!exists.exists) {
        showToast('路径不存在', 'error')
        return
      }
      this.analyzing = true
      try {
        const result = await apiAnalyzeProject(this.form.path, this.form.packageName)
        if (result) {
          this.form.name = result.name || this.form.name
          this.form.packageName = result.name || this.form.name
          this.form.type = result.type || 'other'
          this.form.commands = result.commands || []
          this.form.subModules = result.subModules || []
          if (result.commands && result.commands.length > 0) {
            showToast('已自动识别项目类型: ' + this.getTypeLabel(this.form.type), 'success')
          } else {
            showToast('未能识别项目类型，请手动配置', 'warning')
          }
          if (result.subModules && result.subModules.length > 0) {
            showToast('发现 ' + result.subModules.length + ' 个子模块', 'success')
          }
        }
      } catch (e) {
        showToast('分析项目失败: ' + e.message, 'error')
      }
      this.analyzing = false
    },
    async onPathChange() {
      if (this.form.path && this.form.path.length > 3) {
        await this.analyzeProject()
      }
    },
    saveProject() {
      if (!this.form.name || !this.form.path) {
        showToast('请填写项目名称和路径', 'warning')
        return
      }
      if (this.editingProject) {
        const idx = this.projects.findIndex(p => p.id === this.editingProject.id)
        if (idx !== -1) {
          this.projects[idx] = { ...this.editingProject, ...this.form }
        }
        showToast('项目已更新', 'success')
      } else {
        const newProject = {
          id: randomStr(),
          ...this.form,
          createdAt: Date.now()
        }
        this.projects.push(newProject)
        showToast('项目已添加', 'success')
      }
      this.saveProjects()
      this.closeModal()
    },
    async deleteProject(project) {
      if (confirm('确定要删除项目 "' + project.name + '" 吗？')) {
        this.projects = this.projects.filter(p => p.id !== project.id)
        this.saveProjects()
        showToast('项目已删除', 'success')
      }
    },
    editCommand(index) {
      const cmd = this.form.commands[index]
      this.newCommand = { ...cmd }
      this.editingCommandIndex = index
    },
    addCommand() {
      if (!this.newCommand.name || !this.newCommand.cmd) {
        showToast('请填写命令名称和命令', 'warning')
        return
      }
      if (this.editingCommandIndex >= 0) {
        this.form.commands[this.editingCommandIndex] = { ...this.newCommand }
        this.editingCommandIndex = -1
        showToast('命令已更新', 'success')
      } else {
        this.form.commands.push({ ...this.newCommand })
        showToast('命令已添加', 'success')
      }
      this.newCommand = { name: '', cmd: '', workDir: '', wait: false }
    },
    removeCommand(index) {
      this.form.commands.splice(index, 1)
    },
    removeSubModule(index) {
      this.form.subModules.splice(index, 1)
    },
    addSubModule() {
      if (!this.newSubModule.name) {
        showToast('请填写子模块名称', 'warning')
        return
      }
      if (!this.newSubModule.path) {
        showToast('请填写子模块路径', 'warning')
        return
      }
      if (!this.form.subModules) {
        this.form.subModules = []
      }
      this.form.subModules.push({
        name: this.newSubModule.name,
        path: this.newSubModule.path,
        jarPath: this.newSubModule.jarPath || ''
      })
      this.newSubModule = { name: '', path: '', jarPath: '' }
      showToast('子模块已添加', 'success')
    },
    async startCommand(project, cmd) {
      console.log('startCommand', project, cmd);
      const cmdStr = cmd.cmd
      const workDir = cmd.workDir || project.path
      const runKey = project.id + '_' + cmd.name
      const preCommand = project.preCommand || ''
      const sdkPath = project.sdkPath || ''
      this.runningProjects[runKey] = { ...cmd, pid: Date.now() }
      this.$forceUpdate()
      try {
        const result = await apiRun(cmdStr, workDir, project.type, cmd.name, project.packageName, preCommand, false, sdkPath)
        if (result.success) {
          showToast('命令已启动', 'success')
        } else {
          showToast('启动失败: ' + result.message, 'error')
          delete this.runningProjects[runKey]
          this.$forceUpdate()
        }
      } catch (e) {
        showToast('启动失败: ' + e.message, 'error')
        delete this.runningProjects[runKey]
        this.$forceUpdate()
      }
    },
    async openProjectDir(project) {
      const result = await apiBrowse(project.path)
      if (!result.success) {
        showToast('打开文件夹失败', 'error')
      }
    },
    async showLog(project, content) {
      if (content) {
        this.logContent = content
        this.logFile = project.logFile || ''
        this.showLogPanel = true
        return
      }
      if (!project.logFile) {
        showToast('未配置日志文件', 'warning')
        return
      }
      const exists = await apiExists(project.logFile)
      if (!exists.exists) {
        showToast('日志文件不存在', 'error')
        return
      }
      const result = await apiGetLog(project.logFile)
      this.logContent = result.content || ''
      this.logFile = project.logFile
      this.showLogPanel = true
    },
    closeLogPanel() {
      this.showLogPanel = false
      this.logContent = ''
      this.logFile = ''
    },
    runBackendProject(project) {
      console.log('runBackendProject', project);
      if (!project.subModules || project.subModules.length === 0) {
        showToast('没有可运行的子模块', 'warning')
        return
      }
      if (project.subModules.length === 1) {
        this.runSubModule(project, project.subModules[0])
      } else {
        this.showSubModuleSelector(project)
      }
    },
    showSubModuleSelector(project) {
      this.selectedProject = project
      this.showSubModulePanel = true
    },
    closeSubModulePanel() {
      this.showSubModulePanel = false
      this.selectedProject = null
    },
    async runSubModule(project, mod) {
      let jarPath = mod.jarPath
      if (!jarPath) {
        try {
          const result = await apiFindJar(mod.path)
          jarPath = result.jarPath
        } catch (e) {
          showToast('查找jar文件失败: ' + e.message, 'error')
          return
        }
        if (!jarPath) {
          showToast('未找到jar文件，请先打包项目', 'warning')
          return
        }
      }
      const workDir = mod.path
      const cmdStr = 'java -jar ' + jarPath
      const runKey = project.id + '_' + mod.name
      this.runningProjects[runKey] = { name: mod.name, pid: Date.now() }
      this.$forceUpdate()
      this.closeSubModulePanel()
      try {
        const result = await apiRun(cmdStr, workDir, project.type, mod.name, project.packageName, '', true, project.sdkPath || '')
        if (result.success) {
          showToast('子模块已启动', 'success')
        } else {
          showToast('启动失败: ' + result.message, 'error')
          delete this.runningProjects[runKey]
          this.$forceUpdate()
        }
      } catch (e) {
        showToast('启动失败: ' + e.message, 'error')
        delete this.runningProjects[runKey]
        this.$forceUpdate()
      }
    },
    startStatusCheck() {
      setInterval(() => {
        this.projects.forEach(project => {
          if (project.logFile && this.runningProjects[project.id]) {
            this.checkLogUpdate(project)
          }
        })
      }, 2000)
    },
    async checkLogUpdate(project) {
      if (!project.logFile || !this.showLogPanel) return
      if (this.logFile !== project.logFile) return
      try {
        const result = await apiGetLog(project.logFile)
        if (result.content) {
          this.logContent = result.content
        }
      } catch (e) {}
    },
    exportProjects() {
      exportData(STORAGE_KEY, 'projects.json')
      showToast('导出成功', 'success')
    },
    importProjects() {
      const input = document.createElement('input')
      input.type = 'file'
      input.accept = '.json'
      input.onchange = async (e) => {
        const file = e.target.files[0]
        if (!file) return
        try {
          await importData(STORAGE_KEY, file)
          this.loadProjects()
          showToast('导入成功', 'success')
        } catch (err) {
          showToast('导入失败: ' + err.message, 'error')
        }
      }
      input.click()
    },
    getTypeClass(type) {
      return type === 'backend' ? 'backend' : type === 'frontend' ? 'frontend' : type === 'command' ? 'command' : 'other'
    },
    getTypeLabel(type) {
      return type === 'backend' ? '后' : type === 'frontend' ? '前' : type === 'command' ? '命' : '其他'
    },
    getProjectActions(project) {
      let actions = []
      if (project.commands && project.commands.length > 0) {
        actions = project.commands.map(cmd => ({ type: 'command', data: cmd }))
      }
      if (project.type === 'backend' && project.subModules && project.subModules.length > 0) {
        actions.push({ type: 'run' })
      }
      if (project.logFile) {
        actions.push({ type: 'log' })
      }
      return actions
    },
    toggleViewMode() {
      this.simpleMode = !this.simpleMode
      localStorage.setItem('simpleMode', this.simpleMode)
    },
    copyPath(path) {
      navigator.clipboard.writeText(path).then(() => {
        showToast('路径已复制', 'success')
      }).catch(() => {
        showToast('复制失败', 'error')
      })
    },
    openAddTagModal() {
      this.newTag = { name: '', color: '#409eff' }
      this.showAddTagModal = true
    },
    closeAddTagModal() {
      this.showAddTagModal = false
    },
    addTag() {
      if (!this.newTag.name) {
        showToast('请输入标签名称', 'warning')
        return
      }
      const exists = this.tags.find(t => t.name === this.newTag.name)
      if (exists) {
        showToast('标签名称已存在', 'warning')
        return
      }
      const tag = {
        id: randomStr(),
        name: this.newTag.name,
        color: this.newTag.color
      }
      this.tags.push(tag)
      this.saveProjects()
      showToast('标签已添加', 'success')
      this.closeAddTagModal()
    },
    deleteTag(tagId) {
      const tag = this.tags.find(t => t.id === tagId)
      if (!tag) return
      const referencedProjects = this.projects.filter(p => p.tagIds && p.tagIds.includes(tagId))
      let confirmMsg = '确定要删除标签 "' + tag.name + '" 吗？'
      if (referencedProjects.length > 0) {
        confirmMsg = '标签 "' + tag.name + '" 被 ' + referencedProjects.length + ' 个项目引用，确定要删除吗？\n删除后将取消这些项目与该标签的关联。'
      }
      if (confirm(confirmMsg)) {
        this.tags = this.tags.filter(t => t.id !== tagId)
        this.projects.forEach(p => {
          if (p.tagIds) {
            p.tagIds = p.tagIds.filter(id => id !== tagId)
          }
        })
        if (this.filterTag === tagId) {
          this.filterTag = ''
        }
        this.saveProjects()
        showToast('标签已删除', 'success')
      }
    },
    toggleTag(tagId) {
      if (!this.form.tagIds) {
        this.form.tagIds = []
      }
      const idx = this.form.tagIds.indexOf(tagId)
      if (idx === -1) {
        this.form.tagIds.push(tagId)
      } else {
        this.form.tagIds.splice(idx, 1)
      }
    },
    openAddCommandModal() {
      this.editingCommandProject = null
      this.commandForm = {
        name: '',
        path: '',
        cmd: '',
        tagIds: []
      }
      this.showAddCommandModal = true
    },
    closeAddCommandModal() {
      this.showAddCommandModal = false
      this.editingCommandProject = null
    },
    async browseCommandPath() {
      try {
        const result = await apiSelectDir()
        if (result.path) {
          this.commandForm.path = result.path
        }
      } catch (e) {
        showToast('选择目录失败: ' + e.message, 'error')
      }
    },
    saveCommandProject() {
      if (!this.commandForm.name || !this.commandForm.cmd) {
        showToast('请填写命令名称和命令', 'warning')
        return
      }
      if (this.editingCommandProject) {
        const idx = this.projects.findIndex(p => p.id === this.editingCommandProject.id)
        if (idx !== -1) {
          this.projects[idx] = {
            ...this.editingCommandProject,
            name: this.commandForm.name,
            path: this.commandForm.path || '',
            commands: [{
              name: '运行',
              cmd: this.commandForm.cmd,
              workDir: this.commandForm.path || '',
              wait: false
            }],
            tagIds: this.commandForm.tagIds ? [...this.commandForm.tagIds] : []
          }
        }
        showToast('命令已更新', 'success')
      } else {
        const newProject = {
          id: randomStr(),
          name: this.commandForm.name,
          type: 'command',
          path: this.commandForm.path || '',
          commands: [{
            name: '运行',
            cmd: this.commandForm.cmd,
            workDir: this.commandForm.path || '',
            wait: false
          }],
          tagIds: this.commandForm.tagIds ? [...this.commandForm.tagIds] : [],
          createdAt: Date.now()
        }
        this.projects.push(newProject)
        showToast('命令已添加', 'success')
      }
      this.saveProjects()
      this.closeAddCommandModal()
    },
    toggleCommandTag(tagId) {
      if (!this.commandForm.tagIds) {
        this.commandForm.tagIds = []
      }
      const idx = this.commandForm.tagIds.indexOf(tagId)
      if (idx === -1) {
        this.commandForm.tagIds.push(tagId)
      } else {
        this.commandForm.tagIds.splice(idx, 1)
      }
    }
  }
})

app.mount('#app')
