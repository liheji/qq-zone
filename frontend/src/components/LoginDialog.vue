<template>
  <el-dialog v-model="dialogVisible" :close-on-click-modal="false" :show-close="false" style="min-width: 580px">
    <template #header>
      <el-row justify="space-between">
        下载配置
        <div>
          QQ登录状态：
          <el-tag v-if="loginStatus === 1" size="large" type="success">已登录</el-tag>
          <el-tag v-else size="large" type="primary">未登录</el-tag>
        </div>
      </el-row>
    </template>
    <el-row :gutter="20">
      <el-col :span="16">
        <el-form label-width="auto" style="padding: 20px" ref="dataFormRef" :model="dataForm" :rules="dataRules">
          <el-form-item label="QQ" label-position="right" prop="qq">
            <dynamic-item>
              <el-input v-model="dataForm.qq" placeholder="123456"/>
              <template #suffix>
                <el-popconfirm
                  v-if="loginDisable"
                  confirm-button-text="是"
                  cancel-button-text="否"
                  title="是否重新登录？"
                  @confirm="login">
                  <template #reference>
                    <el-button :icon="ArrowRightBold" :disabled="!loginDisable"></el-button>
                  </template>
                </el-popconfirm>
                <el-button v-else :icon="ArrowRightBold" @click="login" :disabled="loginDisable"
                           title="点击获取登录二维码"/>
              </template>
            </dynamic-item>
          </el-form-item>

          <el-form-item label="好友" label-position="right" prop="friendQQ" title="好友相册[需有访问权限]">
            <dynamic-item>
              <el-select-v2 v-model="dataForm.friendQQ" :options="friendOptions" multiple clearable filterable
                            :disabled="!friendEnable"
                            collapse-tags
                            collapse-tags-tooltip :props="{ value: 'uin', label: 'name' }"
                            placeholder="登录以选择好友[需有访问权限]">
                <template #default="{ item }">
                  <div style="display: flex;align-items: center;">
                    <el-avatar shape="square" :size="30" :src="BASE_URL + item.img"/>
                    <span style="margin-left: 15px;">{{ `${item.name}(${item.uin})` }}</span>
                  </div>
                </template>
              </el-select-v2>
              <template #suffix>
                <el-button :icon="ArrowRightBold" @click="albumList" title="点击获取相册列表"/>
              </template>
            </dynamic-item>
          </el-form-item>
          <el-form-item label="相册" label-position="right" prop="album">
            <el-select-v2 v-model="dataForm.album" :options="albumOptions" :disabled="!albumEnable" multiple clearable
                          filterable collapse-tags
                          collapse-tags-tooltip :props="{ value: 'name', label: 'name' , options: 'albumList'}"
                          placeholder="登录以选择相册">
              <template #default="{ item }">
                <div v-if="item.allowAccess === 1" style="display: flex;align-items: center;">
                  <el-avatar shape="square" :size="30" :src="BASE_URL + item.pre"/>
                  <span style="margin-left: 15px;">{{ item.name }}</span>
                </div>
                <el-tooltip v-else content="无权访问该相册" placement="right" :show-after="200" :hide-after="0"
                            effect="light">
                  <div style="display: flex;align-items: center;">
                    <span style="margin-left: 15px;">{{ item.name }}</span>
                  </div>
                </el-tooltip>
              </template>
            </el-select-v2>
          </el-form-item>
        </el-form>
      </el-col>
      <el-col :span="8" style="padding-top: 10px;">
        <el-image style="width: 150px; height: 150px" :src="BASE_URL + qrCodeUrl">
          <template #error>
            <img src="https://cube.elemecdn.com/e/fd/0fc7d20532fdaf769a25683617711png.png" alt="" width="150"
                 height="150">
          </template>
        </el-image>
      </el-col>
    </el-row>
    <template #footer>
      <div class="dialog-footer">
        <el-popconfirm
          confirm-button-text="是"
          cancel-button-text="否"
          title="是否终止已开始的登录流程?"
          @confirm="cancelLogin"
        >
          <template #reference>
            <el-button type="danger">终止</el-button>
          </template>
        </el-popconfirm>
        <el-button @click="dialogVisible=false">关闭</el-button>
        <el-button type="primary" @click="download">确认</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import QWebSocket from "@/utils/websocket.js";
import {Notify} from "@/utils/notify.js";
import {ArrowRightBold} from "@element-plus/icons-vue";
import {onMounted, reactive, ref, watch} from "vue";
import DynamicItem from "@/components/DynamicItem.vue";
import {listToMap} from "@/utils/tools.js";
import BASE_URL from "@/utils/base.js";

// Props
const props = defineProps({
  visible: Boolean,
  readyState: Number,
  currentQQ: String,
  friendData: Object,
  albumData: Object,
});

// Emits
const emit = defineEmits(["update:visible", "update:friendData", "update:albumData", "update:currentQQ"]);

// Dialog visibility
const dialogVisible = ref(props.visible);
watch(() => props.visible, (val) => (dialogVisible.value = val));
watch(dialogVisible, (val) => emit("update:visible", val));

const readyState = ref(props.readyState);
watch(() => props.readyState, (val) => (readyState.value = val));

// Form states
const dataFormRef = ref();
const albumEnable = ref(false);
const albumOptions = reactive([]);
const friendEnable = ref(false);
const friendOptions = reactive([]);

const friendMap = reactive({});
watch(friendMap, (val) => {
  if (Object.keys(val).length < 0) {
    return
  }
  emit("update:friendData", val)
}, {immediate: true, deep: true});

const albumMap = reactive({});
watch(albumMap, (val) => {
  if (Object.keys(val).length < 0) {
    return
  }
  emit("update:albumData", val)
}, {immediate: true, deep: true});

const dataRules = reactive({
  qq: [{required: true, message: "请输入QQ号", trigger: "blur"}],
  album: [{required: true, message: "请选择相册", trigger: "blur"}],
  friendQQ: [{required: true, message: "请选择好友", trigger: "blur"}],
});

const dataForm = reactive({qq: "", friendQQ: [], album: []});
watch(() => dataForm.qq, (val) => emit("update:currentQQ", val));

const getFormData = () => {
  let albumStr = ""
  const albumList = dataForm.album
  if (albumList instanceof Array) {
    if (albumList.indexOf("All") < 0) {
      albumStr = albumList.join('$$')
    }
  }

  let friendQQStr = ""
  const friendQQList = dataForm.friendQQ
  if (friendQQList instanceof Array) {
    if (friendQQList.indexOf("All") < 0) {
      friendQQStr = friendQQList.join('$$')
    }
  }
  return {qq: dataForm.qq, album: albumStr, friendQQ: friendQQStr}
}

// Login states
const qrCodeUrl = ref("");
const loginStatus = ref(0);
const loginDisable = ref(false);

// Methods
const login = async () => {
  await dataFormRef.value.validateField(["qq"], (valid) => {
    if (valid) {
      if (readyState.value !== 1) {
        Notify('登录提示', '服务器未连接', 'error')
        return
      }

      loginDisable.value = true;
      new QWebSocket().send('login', getFormData())
        .then(resp => {
          if (resp.code === 0) {
            qrCodeUrl.value = resp.data
          } else {
            Notify('登录提示', resp.msg, 'error')
            loginDisable.value = false
          }
        })
        .catch(err => {
          Notify('登录提示', err, 'error')

          loginDisable.value = false
        })
    }
  });
};

const cancelLogin = (callback) => {
  if (readyState.value !== 1) {
    Notify('取消提示', '服务器未连接', 'error')
    return
  }

  new QWebSocket().send('cancel', {cancelType: "login"})
    .then(resp => {
      if (resp.code === 0) {
        Notify('取消提示', resp.msg, 'success')
        if (callback && typeof callback === 'function') {
          callback()
        }
      } else {
        Notify('取消提示', resp.msg, 'error')
        if (callback && typeof callback === 'function') {
          callback()
        }
      }
    })
    .catch(err => {
      Notify('取消提示', err, 'error')
    })
};

const albumList = () => {
  if (readyState.value !== 1) {
    Notify('获取相册提示', '服务器未连接', 'error')
    return
  }

  new QWebSocket().send('album', getFormData())
    .then(resp => {
      if (resp.code === 0) {
        Notify('获取相册提示', "获取成功", 'success')

        Object.assign(albumMap, resp.data)
        albumOptions.length = 0;
        for (const item of friendOptions) {
          const fqq = item.uin;
          if (!albumMap.hasOwnProperty(fqq)) {
            continue
          }

          albumOptions.push({
            name: friendMap[fqq].name,
            albumList: albumMap[fqq]
          })
        }

        albumEnable.value = true
      } else {
        Notify('获取相册提示', resp.msg, 'error')
      }
    })
    .catch(err => {
      Notify('获取相册提示', err, 'error')
    })
};

// Methods
const download = async () => {
  await dataFormRef.value.validateField(["qq", "album"], (valid) => {
    if (valid) {
      if (readyState.value !== 1) {
        Notify('下载提示', '服务器未连接', 'error')
        return
      }

      new QWebSocket().send('download', getFormData())
        .then(resp => {
          if (resp.code === 0) {
            dialogVisible.value = false
          } else {
            Notify('下载提示', resp.msg, 'error')
          }
        })
        .catch(err => {
          Notify('下载提示', err, 'error')
        })
    }
  });
};

const reSetLogin = () => {
  // 重置
  friendOptions.length = 0;
  albumOptions.length = 0;
  Object.assign(friendMap, {})
  Object.assign(albumMap, {})

  qrCodeUrl.value = ""
  albumEnable.value = false
  loginStatus.value = 0
  loginDisable.value = false
  friendEnable.value = false
}

watch(readyState, (newVal, oldVal) => {
  initCurrentWebSocket()
})

onMounted(() => {
  initCurrentWebSocket()
})

const initCurrentWebSocket = () => {
  if (readyState.value !== 1) {
    return
  }

  const obj = new QWebSocket();
  // 监听登录状态
  obj.on('login', resp => {
    Notify('登录提示', resp, 'info')
  })

  obj.on('loginSuccess', resp => {
    Notify('登录提示', resp.msg, 'success')
    // 获取好友列表
    const friendList = resp.friendList
    Object.assign(friendMap, listToMap(friendList, 'uin'))
    friendOptions.length = 0;
    friendOptions.push(...friendList);

    qrCodeUrl.value = ""
    loginStatus.value = 1
    loginDisable.value = false
    friendEnable.value = true
  })

  obj.on('loginError', resp => {
    Notify('登录提示', resp, 'error')

    if (resp.indexOf("取消") < 0) {
      reSetLogin()
    }
  })
}

</script>
