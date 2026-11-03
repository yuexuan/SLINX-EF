<template>
    <button class="theme-switch" :aria-label="isDark ? '切换亮色主题' : '切换暗色主题'" @click="toggleTheme">
        <span class="icon">{{ isDark ? 'light_mode' : 'dark_mode' }}</span>
    </button>
</template>
<script setup lang="ts">
import { isDark } from '@/composable/theme'
onMounted(() => document.body.classList.toggle('light-mode', !isDark.value))
function toggleTheme() {
    isDark.value = !isDark.value
    document.body.classList.toggle('light-mode', !isDark.value)
    localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
</script>
<style scoped>
.theme-switch { padding: 10px; border-radius: 50%; background: var(--color-bg-dark); box-shadow: var(--box-shadow); margin-left: auto; }
.theme-switch:hover { background: var(--color-primary); }
</style>
