<template>
  <div ref="container" class="item-container">
    <div class="item-content" :style="{ width: contentWidth + 'px' }">
      <slot></slot>
    </div>
    <div ref="suffix" class="item-suffix">
      <slot name="suffix"></slot>
    </div>
  </div>
</template>

<script setup>
import {nextTick, onBeforeUnmount, onMounted, ref} from "vue";

const container = ref();
const suffix = ref();
const contentWidth = ref(0);

const updateContentWidth = () => {
  nextTick(() => {
    const containerEl = container.value;
    const suffixEl = suffix.value;
    if (containerEl && suffixEl) {
      const containerWidth = containerEl.offsetWidth;
      const suffixWidth = suffixEl.offsetWidth;
      contentWidth.value = containerWidth - suffixWidth; // 减去按钮宽度和间距
    }
  });
};

onMounted(() => {
  updateContentWidth();
  window.addEventListener("resize", updateContentWidth);
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", updateContentWidth);
});
</script>

<style scoped>
.item-container {
  width: 100%;
  display: flex;
  align-items: center;
}
</style>
