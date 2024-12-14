<template>
  <el-container>
    <el-header>
      <el-row justify="space-between">
        <el-button type="primary" @click="dialogVisible = true">重新配置</el-button>
        <el-link :href="BASE_URL +`/result`" :disabled="!downloadEnable" target="_blank"
                 style="margin-left: 10px;">下载结果数据
        </el-link>
        <div>
          服务器连接状态：
          <el-tag v-if="readyState===0" size="large" type="primary">连接中</el-tag>
          <el-tag v-else-if="readyState===1" size="large" type="success">已连接</el-tag>
          <el-tag v-else-if="readyState===2" size="large" type="warning">断开中</el-tag>
          <el-tag v-else size="large" type="danger">已断开</el-tag>
        </div>
      </el-row>
    </el-header>
    <el-main>
      <el-tabs type="border-card">
        <el-tab-pane v-for="(group, fqq) in downloadProcess" :label="`${friendDataMap[fqq]?.name}(${fqq})`">
          <el-scrollbar height="400px">
            <el-collapse>
              <el-collapse-item v-for="(item, album) in group" :name="album">
                <template #title>
                  <el-progress :percentage="Number(((item.sequence || 0) / (item.photoTotal || 1) * 100).toFixed(2))"
                               :text-inside="true" :stroke-width="20" striped>
                    <template #default="{ percentage }" style="mix-blend-mode: difference;">
                      <span style="font-size: 14px">相册：<span style="font-weight: bold;">[{{ item.albumName }}]</span> 正在下载</span>
                      <span style="font-size: 14px">{{ item.sequence || 0 }}/{{ item.photoTotal || 1 }} ({{
                          percentage
                        }}%)</span>
                    </template>
                  </el-progress>
                </template>
                <div>
                  <template v-for="item in downloadError[fqq][album]">
                    <el-alert
                      :title="`QQ(${item.hostUin})的相册[${item.albumName}]第${item.sequence}个文件：${item.errorMsg}，相册名称：${item.albumName}，文件名：${item.photoName}，文件地址：${item.url}`"
                      type="error" :closable="false"/>
                  </template>
                  <el-alert v-show="!downloadError[fqq][album]" title="暂无错误" type="success"/>
                </div>
              </el-collapse-item>
            </el-collapse>
          </el-scrollbar>
        </el-tab-pane>
      </el-tabs>
    </el-main>
  </el-container>
  <LoginDialog :readyState="readyState" v-model:visible="dialogVisible" v-model:currentQQ="currentQQ"
               v-model:friendData="friendDataMap"/>
</template>

<script setup>
import QWebSocket from "@/utils/websocket.js";
import {MsgBoxHtml, Notify} from "@/utils/notify.js";
import LoginDialog from "@/components/LoginDialog.vue";
import {onMounted, reactive, ref} from "vue";
import BASE_URL from "@/utils/base.js";
// 弹窗相关编码
const dialogVisible = ref(false)

const downloadEnable = ref(false)
const currentQQ = ref("")
const friendDataMap = ref({})
const downloadProcess = reactive({});
const downloadError = reactive({});

// 创建连接相关编码
const readyState = ref(0)
const qWebSocket = ref()
onMounted(function () {
  fetch(`${BASE_URL}/token`)
    .then(resp => resp.json())
    .then(data => {
      initWebSocket(data.token)
    })
    .catch(err => {
      Notify('获取Token提示', err, 'error')
    })

  dialogVisible.value = true
})

function initWebSocket(key) {
  const wsUrl = BASE_URL.replace("http", "ws")
  const obj = new QWebSocket(`${wsUrl}/ws/${key}`);
  obj.onopen(() => {
    readyState.value = obj.readyStatus
  })
  obj.onerror(() => {
    readyState.value = obj.readyStatus
  })
  obj.onclose(() => {
    readyState.value = obj.readyStatus
  })

  // 监听下载进度
  obj.on('downloadInfo', resp => {
    Notify("下载信息", resp, 'info')
  })

  // 监听下载进度
  obj.on('downloadProcess', resp => {
    const fqq = resp.hostUin ?? ""
    const key = resp.albumName ?? ""
    // 更新错误信息
    if (!downloadError[fqq]) {
      downloadError[fqq] = {};
    }
    if (resp.errorMsg.length > 0) {
      if (!downloadError[fqq]) {
        downloadError[fqq][key] = []
      }
      downloadError[fqq][key].push(resp)
    }
    // 更新下载进度
    if (!downloadProcess[fqq]) {
      downloadProcess[fqq] = {};
    }
    downloadProcess[fqq][key] = resp
  })

  obj.on('downloadSuccess', resp => {
    let displayMsg = ""
    for (const item of resp) {
      displayMsg += `${item.downloadDate}<br />QQ空间[${item.qq}]相片/视频下载完成，共有${item.total}个文件<br />已保存${item.succTotal}个文件，其中${item.imageTotal}张相片，${item.videoTotal}部视频<br />包含新增${item.addTotal}，失败${item.totalSuccTotal}，已存在${item.repeatTotal}`
      displayMsg += `<br/><hr>`
    }
    MsgBoxHtml('下载信息统计', displayMsg)
    downloadEnable.value = true
  })

  qWebSocket.value = obj
}

</script>

<style scoped>
.el-dialog {
  padding: 20px;
}

.el-progress {
  width: calc(100% - 35px) !important;
}
</style>
