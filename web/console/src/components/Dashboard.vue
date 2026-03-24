<template>
  <section class="console-module-shell onboarding-shell">
    <div class="console-module-summary-card onboarding-hero-card">
      <div class="console-module-hero onboarding-hero">
        <div class="console-module-hero-copy">
          <div class="console-module-eyebrow">组织引导</div>
          <h2 class="console-module-title">{{ organizationTitle }}</h2>
          <p class="console-module-subtitle">
            这个页面不再显示实例统计，而是帮助你在当前组织下完成最小接入闭环：先创建项目，再在项目下创建应用。
          </p>
        </div>
        <div class="onboarding-hero-actions">
          <BButton variant="primary" @click="goCreateProject">创建第一个项目</BButton>
          <BButton variant="outline-primary" :disabled="!canCreateApplication" @click="goCreateApplication">
            创建第一个应用
          </BButton>
        </div>
      </div>
      <div class="onboarding-progress">
        <div class="onboarding-progress-item" :class="{ 'is-complete': hasProjects }">
          <span class="onboarding-progress-step">01</span>
          <div>
            <strong>创建 Project</strong>
            <div>{{ hasProjects ? `已创建 ${projectCount} 个项目` : '当前组织下还没有项目' }}</div>
          </div>
        </div>
        <div class="onboarding-progress-item" :class="{ 'is-complete': hasApplications, 'is-blocked': !hasProjects }">
          <span class="onboarding-progress-step">02</span>
          <div>
            <strong>创建 Application</strong>
            <div>{{ hasApplications ? `已创建 ${applicationCount} 个应用` : hasProjects ? '项目已就绪，下一步创建应用' : '请先创建项目' }}</div>
          </div>
        </div>
      </div>
    </div>

    <div class="console-module-workspace">
      <aside class="console-module-sidebar">
        <button v-for="item in panels" :key="item.id" type="button" class="console-module-sidebar-link" @click="consoleStore.scrollToPanel(item.id)">
          {{ item.label }}
        </button>
      </aside>

      <div class="console-module-main">
        <div id="dashboard-guide" class="info-card onboarding-card">
          <div class="section-title">开始引导</div>
          <div class="onboarding-step-list">
            <article class="onboarding-step-card" :class="{ 'is-complete': hasProjects }">
              <div class="onboarding-step-heading">
                <span class="onboarding-step-badge">Step 1</span>
                <div>
                  <h3>先创建一个项目</h3>
                  <p>项目是应用、用户授权和后续接入配置的承载单元。新 organization 建好后，第一步应该先把项目结构落下。</p>
                </div>
              </div>
              <div class="onboarding-step-state">
                <strong>{{ hasProjects ? '已完成' : '待完成' }}</strong>
                <span>{{ hasProjects ? `当前共有 ${projectCount} 个项目` : '还没有任何项目' }}</span>
              </div>
              <div class="onboarding-step-actions">
                <BButton variant="primary" @click="goCreateProject">{{ hasProjects ? '继续创建项目' : '创建项目' }}</BButton>
                <BButton variant="outline-secondary" :disabled="!hasProjects" @click="goProjectList">查看项目列表</BButton>
              </div>
            </article>

            <article class="onboarding-step-card" :class="{ 'is-complete': hasApplications, 'is-blocked': !hasProjects }">
              <div class="onboarding-step-heading">
                <span class="onboarding-step-badge">Step 2</span>
                <div>
                  <h3>在项目下创建应用</h3>
                  <p>应用定义 OAuth/OIDC 接入方式、回调地址、Grant Type 和令牌参数。没有应用，组织还不能真正接入业务系统。</p>
                </div>
              </div>
              <div class="onboarding-step-state">
                <strong>{{ hasApplications ? '已完成' : hasProjects ? '待完成' : '被阻塞' }}</strong>
                <span>{{ hasApplications ? `当前共有 ${applicationCount} 个应用` : hasProjects ? '请选择一个项目作为应用承载容器' : '请先完成 Step 1' }}</span>
              </div>
              <div v-if="hasProjects" class="onboarding-project-selector">
                <label class="form-label">目标项目</label>
                <BFormSelect v-model="selectedProjectId" :options="projectOptions" />
              </div>
              <div class="onboarding-step-actions">
                <BButton variant="primary" :disabled="!canCreateApplication" @click="goCreateApplication">
                  {{ hasApplications ? '继续创建应用' : '创建应用' }}
                </BButton>
                <BButton variant="outline-secondary" :disabled="!selectedProjectId" @click="goSelectedProjectDetail">查看项目详情</BButton>
              </div>
            </article>
          </div>
        </div>

        <div id="dashboard-workspace" class="info-card onboarding-card">
          <div class="section-title">当前组织工作区</div>
          <div class="onboarding-summary-grid">
            <div class="onboarding-summary-tile">
              <span>当前组织</span>
              <strong>{{ organizationTitle }}</strong>
            </div>
            <div class="onboarding-summary-tile">
              <span>项目数量</span>
              <strong>{{ projectCount }}</strong>
            </div>
            <div class="onboarding-summary-tile">
              <span>应用数量</span>
              <strong>{{ applicationCount }}</strong>
            </div>
            <div class="onboarding-summary-tile">
              <span>下一步</span>
              <strong>{{ nextActionLabel }}</strong>
            </div>
          </div>
          <div v-if="projects.length" class="onboarding-project-grid">
            <button
              v-for="project in projects"
              :key="project.id"
              type="button"
              class="onboarding-project-card"
              :class="{ 'is-active': selectedProjectId === project.id }"
              @click="selectedProjectId = project.id"
            >
              <div class="onboarding-project-card-top">
                <strong>{{ project.name || project.id }}</strong>
                <span>{{ project.applications?.length ?? 0 }} apps</span>
              </div>
              <div class="onboarding-project-card-id">{{ project.id }}</div>
              <div class="onboarding-project-card-meta">
                <span>{{ project.description || '当前项目还没有描述。' }}</span>
              </div>
            </button>
          </div>
          <div v-else class="onboarding-empty-state">
            <strong>当前 organization 还是空白的。</strong>
            <span>从创建第一个项目开始，后续再补应用和接入配置。</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch, watchEffect } from 'vue'
import { useRouter } from 'vue-router'
import { BButton, BFormSelect } from 'bootstrap-vue-next'
import { useConsoleStore } from '../stores/console'
import { useOrganizationStore } from '../stores/organization'
import { useProjectStore } from '../stores/project'

const router = useRouter()
const consoleStore = useConsoleStore()
const organizationStore = useOrganizationStore()
const projectStore = useProjectStore()

const selectedProjectId = ref('')

const panels = [
  { id: 'dashboard-guide', label: '开始引导' },
  { id: 'dashboard-workspace', label: '当前组织工作区' }
]

const organizationId = computed(() => consoleStore.currentOrganizationId || organizationStore.currentOrganization?.id || '')
const projects = computed(() => projectStore.projects)
const projectCount = computed(() => projects.value.length)
const applicationCount = computed(() => projects.value.reduce((total: number, item: any) => total + (item.applications?.length ?? 0), 0))
const hasProjects = computed(() => projectCount.value > 0)
const hasApplications = computed(() => applicationCount.value > 0)
const canCreateApplication = computed(() => Boolean(selectedProjectId.value))
const organizationTitle = computed(() => organizationStore.currentOrganization?.name || organizationStore.currentOrganization?.id || '当前组织')
const projectOptions = computed(() => projects.value.map((item: any) => ({ value: item.id, text: item.name || item.id })))
const nextActionLabel = computed(() => {
  if (!hasProjects.value) return '创建项目'
  if (!hasApplications.value) return '创建应用'
  return '继续完善接入配置'
})

watchEffect(() => {
  consoleStore.setPageHeader('组织引导', '在当前 organization 下逐步创建 project 和 application，完成最小接入闭环。')
})

watch(
  organizationId,
  async (value) => {
    if (!value) {
      return
    }
    await projectStore.loadProjects(value)
    if (!projects.value.some((item: any) => item.id === selectedProjectId.value)) {
      selectedProjectId.value = projects.value[0]?.id ?? ''
    }
  },
  { immediate: true }
)

async function goCreateProject() {
  if (!organizationId.value) {
    await router.push({ name: 'console-organization-manage' })
    return
  }
  await router.push({
    name: 'console-project-list',
    params: { organizationId: organizationId.value },
    query: { create: 'project' }
  })
}

async function goProjectList() {
  if (!organizationId.value) {
    return
  }
  await router.push({ name: 'console-project-list', params: { organizationId: organizationId.value } })
}

async function goCreateApplication() {
  if (!organizationId.value || !selectedProjectId.value) {
    return
  }
  await router.push({
    name: 'console-project-detail',
    params: { organizationId: organizationId.value, projectId: selectedProjectId.value },
    query: { create: 'application', projectId: selectedProjectId.value }
  })
}

async function goSelectedProjectDetail() {
  if (!organizationId.value || !selectedProjectId.value) {
    return
  }
  await router.push({
    name: 'console-project-detail',
    params: { organizationId: organizationId.value, projectId: selectedProjectId.value }
  })
}
</script>

<style scoped>
.onboarding-shell {
  gap: 1.5rem;
}

.onboarding-hero-card {
  overflow: hidden;
  background:
    radial-gradient(circle at top right, rgba(12, 110, 253, 0.16), transparent 28rem),
    linear-gradient(135deg, #ffffff 0%, #f4f8ff 100%);
}

.onboarding-hero {
  align-items: flex-start;
  gap: 1rem;
}

.onboarding-hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.onboarding-progress {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(16rem, 1fr));
  gap: 1rem;
  margin-top: 1.5rem;
}

.onboarding-progress-item {
  display: flex;
  gap: 0.875rem;
  padding: 1rem 1.125rem;
  border-radius: 1rem;
  background: rgba(255, 255, 255, 0.78);
  border: 1px solid rgba(15, 23, 42, 0.08);
}

.onboarding-progress-item.is-complete {
  border-color: rgba(25, 135, 84, 0.35);
  background: rgba(25, 135, 84, 0.08);
}

.onboarding-progress-item.is-blocked {
  opacity: 0.65;
}

.onboarding-progress-step {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 999px;
  background: #0d6efd;
  color: #fff;
  font-weight: 700;
}

.onboarding-card {
  padding: 1.5rem;
}

.onboarding-step-list {
  display: grid;
  gap: 1rem;
}

.onboarding-step-card {
  display: grid;
  gap: 1rem;
  padding: 1.25rem;
  border-radius: 1rem;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: #fff;
}

.onboarding-step-card.is-complete {
  border-color: rgba(25, 135, 84, 0.35);
  background: linear-gradient(180deg, rgba(25, 135, 84, 0.05), #fff);
}

.onboarding-step-card.is-blocked {
  opacity: 0.75;
}

.onboarding-step-heading {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 1rem;
  align-items: start;
}

.onboarding-step-heading h3 {
  margin: 0 0 0.35rem;
  font-size: 1.125rem;
}

.onboarding-step-heading p {
  margin: 0;
  color: #64748b;
}

.onboarding-step-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 4.5rem;
  height: 2rem;
  padding: 0 0.9rem;
  border-radius: 999px;
  background: #e7f1ff;
  color: #0d6efd;
  font-weight: 700;
}

.onboarding-step-state {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
  color: #475569;
}

.onboarding-step-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.onboarding-project-selector {
  max-width: 24rem;
}

.onboarding-summary-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
  gap: 1rem;
}

.onboarding-summary-tile {
  display: grid;
  gap: 0.45rem;
  padding: 1rem;
  border-radius: 1rem;
  background: #f8fafc;
  border: 1px solid rgba(15, 23, 42, 0.06);
}

.onboarding-summary-tile span {
  color: #64748b;
  font-size: 0.875rem;
}

.onboarding-summary-tile strong {
  font-size: 1.1rem;
}

.onboarding-project-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(16rem, 1fr));
  gap: 1rem;
  margin-top: 1.25rem;
}

.onboarding-project-card {
  display: grid;
  gap: 0.7rem;
  padding: 1rem;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 1rem;
  background: #fff;
  text-align: left;
  transition: border-color 0.2s ease, transform 0.2s ease, box-shadow 0.2s ease;
}

.onboarding-project-card:hover,
.onboarding-project-card.is-active {
  border-color: rgba(13, 110, 253, 0.35);
  box-shadow: 0 14px 30px rgba(15, 23, 42, 0.08);
  transform: translateY(-2px);
}

.onboarding-project-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.onboarding-project-card-top span,
.onboarding-project-card-meta,
.onboarding-project-card-id {
  color: #64748b;
  font-size: 0.875rem;
}

.onboarding-empty-state {
  display: grid;
  gap: 0.4rem;
  margin-top: 1.25rem;
  padding: 1.25rem;
  border-radius: 1rem;
  background: #f8fafc;
  color: #475569;
}

@media (max-width: 767px) {
  .onboarding-step-heading {
    grid-template-columns: 1fr;
  }
}
</style>
