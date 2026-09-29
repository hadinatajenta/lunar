<script setup lang="ts">
import PageLayout from "../../../components/layout/PageLayout.vue"
import ChatSidebar from "../components/ChatSidebar.vue"
import ChatWindow from "../components/ChatWindow.vue"
import DomainToggleModal from "../components/DomainToggleModal.vue"
</script>

<template>
  <PageLayout>
    <div class="copilot-container">
      <div class="copilot-grid">
        <ChatSidebar class="copilot-sidebar-col" />
        <ChatWindow class="copilot-main-col" />
      </div>
      <DomainToggleModal />
    </div>
  </PageLayout>
</template>

<style scoped>
.copilot-container {
  height: calc(100vh - 64px);
  overflow: hidden;
}

.copilot-grid {
  display: grid;
  grid-template-columns: 240px 1fr;
  height: 100%;
}

@media (max-width: 768px) {
  .copilot-grid {
    grid-template-columns: 1fr;
  }
  .copilot-sidebar-col {
    display: none;
  }
}
</style>
