import {ElMessageBox, ElNotification} from "element-plus";

const Notify = (() => {
  let preNotify = null
  return (title, message, type) => {
    if (preNotify) {
      preNotify.close()
    }
    preNotify = ElNotification({title: title, message: message, type: type})
  };
})()

const MsgBoxHtml = (() => {
  let preAlert = null
  return (title, message) => {
    if (preAlert) {
      preAlert.close()
    }
    preAlert = ElMessageBox.alert(message, title, {dangerouslyUseHTMLString: true})
  };
})()

export {Notify, MsgBoxHtml};
