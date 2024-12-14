const listToMap = (dataList, key) => {
  const map = {};
  dataList.forEach(data => {
    map[data[key]] = data;
  })
  return map;
}

const copyMapInto = (source, target) => {
  if (source === null || typeof source !== 'object' ||
    target === null || typeof target !== 'object') {
    return
  }

  // 删除原来的
  Object.assign(target, {});

  // 添加新的
  const keyList = Object.keys(source);
  for (const key of keyList) {
    target[key] = source[key];
  }
}

function keySortList(sortKeyList) {
  let keyList = sortKeyList;
  return (object) => {
    if (keyList == null || object == null || typeof object !== 'object') {
      return [];
    }
    return keyList.filter(key => key in object);
  }
}

export {
  listToMap,
  copyMapInto,
  keySortList,
}
