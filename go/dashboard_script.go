package main

// dashboardScript is the browser logic of the dashboard resource. It is kept
// apart from the markup so neither file grows unwieldy, and it must stay free of
// backticks because it is embedded in a Go raw string literal.
const dashboardScript = `
(function () {
  var base = "/v0/management/apikey-quota";
  // hostBase is the host's own management API, used to rotate client API keys.
  var hostBase = "/v0/management";
  var KEY_STORAGE = "apikey-quota-mgmt-key";
  var LANG_STORAGE = "apikey-quota-lang";

  var I18N = {
    en: {
      title: "API Key Quota",
      subtitle: "Usage and limits per client API key, served by the apikey-quota plugin.",
      mgmtKey: "Management key",
      mgmtKeyHint: "Required, same key as the control panel",
      show: "Show",
      hide: "Hide",
      language: "Language",
      autoRefresh: "Auto refresh",
      off: "off",
      reload: "Reload",
      tabKeys: "Keys",
      colKey: "Key",
      colActive: "Live",
      colDaily: "Daily",
      colMonthly: "Monthly",
      colTotal: "Total",
      colModels: "Blocked models",
      colBlocked: "Rejected",
      colLastUsed: "Last used",
      colPattern: "Pattern",
      colModel: "Model",
      colMatched: "Matched rule",
      colTokens: "Tokens",
      colRequests: "Requests",
      colCost: "Cost",
      globalBlocked: "Globally blocked models",
      globalBlockedHint: "One glob pattern per line, for example gpt-4* or claude-*-opus*. Every API key is refused these models with 403.",
      priceTable: "Price table",
      priceHint: "USD per one million tokens. The longest matching pattern wins; the default applies when nothing matches.",
      defaultPrice: "Default price",
      pInput: "Input",
      pOutput: "Output",
      pReasoning: "Reasoning",
      pCacheRead: "Cache read",
      pCacheWrite: "Cache write",
      observed: "Recognized models",
      observedHint: "Models the plugin has actually billed, with the price rule each one resolved to.",
      addRow: "Add model",
      useConfig: "Use configuration",
      save: "Save",
      cancel: "Cancel",
      remove: "Remove",
      clearOverrides: "Clear overrides",
      editHint: "Blank or 0 inherits the configured limit. Use -1 for unlimited.",
      keyBlocked: "Blocked models for this key",
      disableKey: "Requests are rejected with 403 until the key is enabled again.",
      limCost: "Budget (USD)",
      limTokens: "Tokens",
      limRequests: "Requests",
      limInherit: "Inherit configuration",
      note: "Note",
      phNote: "free-form remark, for example the team or service using this key",
      modelsBtn: "Models",
      modelsTitle: "Model usage",
      modelsHint: "Totals this key has accumulated per model, across the whole tracking history.",
      colShare: "Share",
      totalRow: "Total",
      noModelUsage: "This key has not been billed for any model yet.",
      close: "Close",
      addBtn: "Add",
      removeTag: "Remove",
      noBlocked: "No model is blocked for this key.",
      phBlocked: "one pattern per line",
      phBlockedOne: "model name or pattern, e.g. gpt-4*",
      pauseBtn: "Pause",
      enableBtn: "Enable",
      paused: "paused",
      inUse: "in use",
      unpriced: "unpriced",
      srcBuiltin: "built-in default prices",
      srcConfig: "prices from configuration",
      srcRuntime: "prices overridden at runtime",
      cardKeys: "Tracked keys",
      cardActive: "Active requests",
      cardTokens: "Total tokens",
      cardRequests: "Total requests",
      cardCost: "Total cost",
      cardBlocked: "Rejected requests",
      limitsBtn: "Edit",
      resetBtn: "Reset",
      noUsage: "No API key usage recorded yet.",
      noModels: "No model has been billed yet.",
      noPrices: "No per-model price configured.",
      enterKey: "Set the management key to load usage.",
      keyRequired: "The management key is required, including on localhost.",
      authFailed: "Authentication failed. Check the management key in Settings.",
      loading: "Loading...",
      updated: "Updated",
      enforceOn: "enforcement on",
      enforceOff: "enforcement off",
      sent: "sent",
      chars: "characters",
      refreshStopped: "auto refresh stopped",
      failed: "Failed",
      resetFailed: "Reset failed",
      saveFailed: "Save failed",
      saved: "Saved",
      fromConfig: "from configuration",
      customValues: "runtime override active",
      defaultRule: "default",
      unlimited: "no limit",
      none: "none",
      disabled: "disabled",
      overridden: "override",
      blocked: "blocked",
      inherited: "inherited",
      resetHint: "Clears the accumulated counters of the selected period. This cannot be undone.",
      resetPeriod: "Period",
      resetAll: "Everything, including the model breakdown and reject counter",
      mappings: "Model mappings",
      mappingsHint: "Rewrites a requested model to another model before routing. Patterns use globs such as gpt-4o*. The first matching rule wins, and a key's own rules are checked before these. Leave the provider blank to infer it from the target model.",
      colFrom: "Requested model",
      colTo: "Mapped to",
      colProvider: "Provider",
      addRule: "Add rule",
      noMappings: "No model mapping configured.",
      keyMappings: "Model mappings for this key",
      noKeyMappings: "This key has no mapping of its own; global rules still apply.",
      phMapFrom: "requested model",
      phMapTo: "target model",
      phMapProvider: "provider",
      autoProvider: "auto",
      providersSeen: "Providers with credentials",
      providersUnknown: "Available providers appear here once a request has been routed.",
      mappedPill: "mapped",
      fallbacks: "Fallback models",
      fallbacksHint: "One model per line, tried in order when the requested model fails upstream with a timeout, rate limit, overload, or server error. Applies to keys without a list of their own. Token counting and streams that already started are never retried.",
      keyFallbacks: "Fallback models for this key",
      fallbackOwn: "Use a fallback list of its own",
      fallbackOwnHint: "One model per line, tried in order. Leave it empty to turn fallback off for this key.",
      fallbackInherited: "Follows the global list",
      fallbackNone: "No global fallback models, so failures are returned as they are.",
      phFallbacks: "gpt-5-mini\nclaude-haiku-4-5",
      fallbackPill: "fallback",
      mapIncomplete: "Both the requested model and the target model are required.",
      tabRules: "Model rules",
      tabPricing: "Pricing",
      settings: "Settings",
      settingsHint: "The key is kept in this browser tab only and is sent with every request.",
      connect: "Connect",
      setKey: "Set management key",
      colActions: "Actions",
      phFilter: "Filter by key, note or id",
      filterInfo: "shown",
      noMatch: "No key matches the filter.",
      pauseKey: "Pause this key",
      limitsSection: "Limits",
      editKey: "Edit key",
      pickModel: "Select requested model",
      customPattern: "Custom pattern…",
      phMapFromCustom: "model or pattern, e.g. gpt-4o*",
      rotateBtn: "Rotate",
      rotateTitle: "Rotate key",
      rotateHint: "Replace this key with a newly generated random key.",
      rotateWarnOld: "The old key stops working immediately; clients using it must switch to the new key.",
      rotateWarnMove: "Usage, limits, note, blocked models and mappings move to the new key.",
      rotateWarnOnce: "The new key is shown only once. Copy it right away.",
      rotateGo: "Generate and replace",
      rotating: "Rotating...",
      newKey: "New key",
      copy: "Copy",
      copied: "Copied",
      rotateDone: "I have saved it",
      retryMigrate: "Retry migration",
      rotateNotFound: "This key is not in the CPA api-keys list, so it cannot be rotated here.",
      rotateHostFailed: "Replacing the key in CPA failed",
      rotateMigrateFailed: "The new key is active, but moving usage and settings failed",
      rotateConfigured: "This key also has an entry under keys in the plugin configuration. Its settings now apply to the new key as runtime overrides; update that entry's key to the new key, or remove the entry.",
      rotated: "Key rotated",
      deleteBtn: "Delete",
      deleteTitle: "Delete keys",
      deleteHint: "Removes the usage records, limits, note, blocked models and mappings. This cannot be undone.",
      deleteFromHost: "Also remove from CPA api-keys",
      deleteFromHostHint: "The key stops working immediately. Without this, only the record here is cleared, and the key is tracked again on its next request.",
      deleting: "Deleting...",
      deleted: "Keys deleted",
      deleteFailed: "Delete failed",
      deleteHostFailed: "removing it from CPA failed, so its record was kept",
      deleteNotInHost: "not in the CPA api-keys list; only the record here was cleared",
      deleteConfigured: "still has an entry under keys in the plugin configuration, so it stays listed; remove that entry by hand",
      selectAll: "Select all shown keys",
      selectRow: "Select this key",
      clearSelection: "Clear selection",
      selectedCount: "Selected",
      keysUnit: "keys",
      batchSaved: "Applied to the selected keys",
      tagConc: "Conc",
      rateSection: "Rate limits",
      rateHint: "Measured over the last minute. Blank or 0 inherits the configured value. Use -1 for unlimited.",
      rateRPM: "Requests per minute",
      rateTPM: "Tokens per minute",
      rateConc: "Concurrent requests",
      schedSection: "Schedule",
      schedOwn: "Use a schedule of its own",
      schedFromRuntime: "Set in this dashboard. Turn it off to fall back to the configuration.",
      schedFromConfig: "Now following this key's schedule in the plugin configuration",
      schedFromDefault: "Now following default_schedule of the plugin configuration",
      schedNone: "No schedule applies: the key may be used at any time.",
      schedWindows: "windows",
      schedHint: "The first window covering a moment applies, so list narrow windows such as a lunch break before wide ones. An end at or before the start runs past midnight; equal times cover the whole day. Window quotas restart with every occurrence.",
      schedOutside: "Outside every window",
      schedAllow: "Allow, with the normal limits",
      schedBlockAll: "Refuse the key",
      addWindow: "Add window",
      winName: "Window name",
      moveUp: "Move up",
      winLimit: "Use with limits",
      winBlock: "Refuse the key",
      winQuota: "Window quota",
      winRate: "Window rate",
      winNoQuota: "No window quota",
      winNeedDays: "Pick at least one day for every window.",
      winNeedTime: "Every window needs a start and an end time.",
      dayNames: "Mon,Tue,Wed,Thu,Fri,Sat,Sun",
      inWindow: "Window",
      outsideSchedule: "Outside schedule",
      blockedWindow: "Blocked window",
      windowUntil: "until",
      showPie: "Show pie chart",
      metricCost: "Cost",
      metricTokens: "Tokens",
      metricRequests: "Requests",
      otherModels: "Other",
      pieShare: "share",
      pieTooFew: "The pie chart needs at least two models with usage.",
      tabAccounts: "Accounts",
      accountsTitle: "Accounts",
      accountsHint: "Host credentials with the requests they are serving right now, the rule that applies to them, and the keys bound to them.",
      accountsListFailed: "The host did not list its accounts, so only accounts serving requests are shown",
      noAccounts: "No accounts.",
      colAccount: "Account",
      colLive: "Live",
      colRule: "Rule",
      colBoundKeys: "Bound keys",
      colConcurrency: "Concurrency",
      colExclusive: "Exclusive",
      accountRules: "Account rules",
      accountRulesHint: "Matched against the account ID, first matching rule wins. Concurrency caps the requests an account serves at once, blank or 0 for unlimited. An exclusive account only serves the keys bound to it.",
      noAccountRules: "No account rules. Every account serves every key without a concurrency cap.",
      exclusivePill: "exclusive",
      unavailablePill: "unavailable",
      disabledPill: "disabled",
      unlimited: "unlimited",
      noRule: "No rule",
      sharedAccount: "Shared",
      rateQueue: "Queue wait (seconds)",
      rateQueueHint: "When the key or every account it may use is at its concurrency limit, a request waits this long for a free slot before it is rejected.",
      keyAccounts: "Accounts for this key",
      keyAccountsHint: "Account IDs or glob patterns. Requests of this key are scheduled on these accounts first. Leave it empty to use every account.",
      noKeyAccounts: "Not bound: the key uses every shared account.",
      phAccount: "codex-team-*.json",
      strictAccounts: "Only use these accounts",
      strictAccountsHint: "When none of them can serve a request, it fails instead of falling back to the shared accounts.",
      boundPill: "accounts",
      strictPill: "strict"
    },
    zh: {
      title: "API 密钥配额",
      subtitle: "由 apikey-quota 插件提供的客户端 API 密钥用量与限额。",
      mgmtKey: "管理密钥",
      mgmtKeyHint: "必填，与控制面板登录使用同一个密钥",
      show: "显示",
      hide: "隐藏",
      language: "语言",
      autoRefresh: "自动刷新",
      off: "关闭",
      reload: "刷新",
      tabKeys: "密钥",
      colKey: "密钥",
      colActive: "实时",
      colDaily: "当日",
      colMonthly: "当月",
      colTotal: "累计",
      colModels: "禁用模型",
      colBlocked: "已拦截",
      colLastUsed: "最近使用",
      colPattern: "匹配规则",
      colModel: "模型",
      colMatched: "命中规则",
      colTokens: "Token",
      colRequests: "请求数",
      colCost: "费用",
      globalBlocked: "全局禁用模型",
      globalBlockedHint: "每行一条通配规则，例如 gpt-4* 或 claude-*-opus*。所有 API 密钥调用这些模型都会返回 403。",
      priceTable: "价格表",
      priceHint: "单位为每百万 Token 的美元价。命中多条规则时最长的规则优先；都不命中则使用默认价格。",
      defaultPrice: "默认价格",
      pInput: "输入",
      pOutput: "输出",
      pReasoning: "推理",
      pCacheRead: "缓存读取",
      pCacheWrite: "缓存写入",
      observed: "已识别模型",
      observedHint: "插件实际计过费的模型，以及每个模型命中的价格规则。",
      addRow: "添加模型",
      useConfig: "恢复为配置文件",
      save: "保存",
      cancel: "取消",
      remove: "删除",
      clearOverrides: "清除覆盖",
      editHint: "留空或填 0 表示继承配置值，填 -1 表示不限。",
      keyBlocked: "该密钥禁用的模型",
      disableKey: "暂停期间该密钥的请求均返回 403，重新启用后恢复。",
      limCost: "预算（美元）",
      limTokens: "Token 数",
      limRequests: "请求数",
      limInherit: "继承配置",
      note: "备注",
      phNote: "自由填写，例如使用该密钥的团队或服务",
      modelsBtn: "模型",
      modelsTitle: "模型用量",
      modelsHint: "该密钥按模型累计的用量，统计自开始跟踪以来的全部历史。",
      colShare: "占比",
      totalRow: "合计",
      noModelUsage: "该密钥尚未对任何模型计费。",
      close: "关闭",
      addBtn: "添加",
      removeTag: "移除",
      noBlocked: "该密钥未禁用任何模型。",
      phBlocked: "每行一条规则",
      phBlockedOne: "模型名或通配规则，例如 gpt-4*",
      pauseBtn: "暂停",
      enableBtn: "启用",
      paused: "已暂停",
      inUse: "使用中",
      unpriced: "无价格",
      srcBuiltin: "使用内置默认价格",
      srcConfig: "价格来自配置文件",
      srcRuntime: "价格已被运行时覆盖",
      cardKeys: "跟踪密钥数",
      cardActive: "当前并发",
      cardTokens: "Token 总量",
      cardRequests: "请求总数",
      cardCost: "总费用",
      cardBlocked: "被拦截请求",
      limitsBtn: "编辑",
      resetBtn: "重置",
      noUsage: "尚未记录任何 API 密钥用量。",
      noModels: "尚未对任何模型计费。",
      noPrices: "未配置任何按模型的价格。",
      enterKey: "请先设置管理密钥以加载用量。",
      keyRequired: "管理密钥是必填的，本机访问同样需要。",
      authFailed: "认证失败，请在设置中检查管理密钥。",
      loading: "加载中...",
      updated: "更新于",
      enforceOn: "限额已启用",
      enforceOff: "限额未启用",
      sent: "已发送",
      chars: "个字符",
      refreshStopped: "自动刷新已停止",
      failed: "失败",
      resetFailed: "重置失败",
      saveFailed: "保存失败",
      saved: "已保存",
      fromConfig: "来自配置文件",
      customValues: "运行时覆盖生效中",
      defaultRule: "默认",
      unlimited: "不限",
      none: "无",
      disabled: "已禁用",
      overridden: "已覆盖",
      blocked: "已禁用",
      inherited: "继承",
      resetHint: "将清空所选周期的累计计数，且无法撤销。",
      resetPeriod: "周期",
      resetAll: "全部，包括模型明细和拦截计数",
      mappings: "模型映射",
      mappingsHint: "在路由之前把请求的模型改写为另一个模型。规则支持通配符，例如 gpt-4o*。按顺序命中第一条规则，且密钥自己的规则优先于这里的全局规则。提供商留空时根据目标模型自动推断。",
      colFrom: "请求模型",
      colTo: "映射为",
      colProvider: "提供商",
      addRule: "添加规则",
      noMappings: "未配置任何模型映射。",
      keyMappings: "该密钥的模型映射",
      noKeyMappings: "该密钥没有自己的映射规则，全局规则仍然生效。",
      phMapFrom: "请求模型",
      phMapTo: "目标模型",
      phMapProvider: "提供商",
      autoProvider: "自动",
      providersSeen: "已配置凭证的提供商",
      providersUnknown: "有请求经过路由后，这里会显示可用的提供商。",
      mappedPill: "已映射",
      fallbacks: "兜底模型",
      fallbacksHint: "每行一个模型，请求的模型在上游超时、限流、过载或服务端出错时，按顺序改用这些模型重试。只作用于没有单独设置兜底列表的密钥。计数 token 的请求和已经开始输出的流式请求不会重试。",
      keyFallbacks: "该密钥的兜底模型",
      fallbackOwn: "为该密钥单独设置兜底模型",
      fallbackOwnHint: "每行一个模型，按顺序尝试。留空表示该密钥不使用兜底。",
      fallbackInherited: "沿用全局兜底列表",
      fallbackNone: "未设置全局兜底模型，失败时直接返回错误。",
      phFallbacks: "gpt-5-mini\nclaude-haiku-4-5",
      fallbackPill: "兜底",
      mapIncomplete: "请求模型和目标模型都必须填写。",
      tabRules: "模型规则",
      tabPricing: "价格",
      settings: "设置",
      settingsHint: "密钥只保存在当前浏览器标签页中，并随每个请求发送。",
      connect: "连接",
      setKey: "设置管理密钥",
      colActions: "操作",
      phFilter: "按密钥、备注或 ID 筛选",
      filterInfo: "已显示",
      noMatch: "没有匹配筛选条件的密钥。",
      pauseKey: "暂停该密钥",
      limitsSection: "限额",
      editKey: "编辑密钥",
      pickModel: "选择请求模型",
      customPattern: "自定义规则…",
      phMapFromCustom: "模型名或通配规则，例如 gpt-4o*",
      rotateBtn: "更换",
      rotateTitle: "更换密钥",
      rotateHint: "生成一个新的随机密钥来替换当前密钥。",
      rotateWarnOld: "旧密钥会立即失效，使用它的客户端需要改用新密钥。",
      rotateWarnMove: "用量、限额、备注、禁用模型和模型映射会迁移到新密钥。",
      rotateWarnOnce: "新密钥只显示这一次，请立即复制保存。",
      rotateGo: "生成并替换",
      rotating: "正在更换...",
      newKey: "新密钥",
      copy: "复制",
      copied: "已复制",
      rotateDone: "我已保存",
      retryMigrate: "重试迁移",
      rotateNotFound: "在 CPA 的 api-keys 列表中找不到这个密钥，无法在这里更换。",
      rotateHostFailed: "在 CPA 中替换密钥失败",
      rotateMigrateFailed: "新密钥已生效，但迁移用量和设置失败",
      rotateConfigured: "该密钥在插件配置的 keys 中也有条目，其设置已作为运行时覆盖应用到新密钥。请把该条目的 key 改为新密钥，或删除该条目。",
      rotated: "密钥已更换",
      deleteBtn: "删除",
      deleteTitle: "删除密钥",
      deleteHint: "删除用量记录、限额、备注、禁用模型和模型映射，且无法恢复。",
      deleteFromHost: "同时从 CPA 的 api-keys 中移除",
      deleteFromHostHint: "密钥会立即失效。不勾选时只清除这里的记录，该密钥下次请求时会重新被统计。",
      deleting: "正在删除...",
      deleted: "密钥已删除",
      deleteFailed: "删除失败",
      deleteHostFailed: "从 CPA 移除失败，已保留其记录",
      deleteNotInHost: "不在 CPA 的 api-keys 列表中，只清除了这里的记录",
      deleteConfigured: "在插件配置的 keys 中仍有条目，因此仍会显示；请手动删除该条目",
      selectAll: "全选当前显示的密钥",
      selectRow: "选择该密钥",
      clearSelection: "取消选择",
      selectedCount: "已选",
      keysUnit: "个密钥",
      batchSaved: "已应用到所选密钥",
      tagConc: "并发",
      rateSection: "速度限制",
      rateHint: "按最近一分钟统计。留空或 0 表示继承配置，-1 表示不限。",
      rateRPM: "每分钟请求数",
      rateTPM: "每分钟 Token 数",
      rateConc: "最大并发数",
      schedSection: "时段规则",
      schedOwn: "为该密钥单独设置时段规则",
      schedFromRuntime: "在面板中设置。关闭后回退到配置文件中的规则。",
      schedFromConfig: "当前使用插件配置中该密钥的时段规则",
      schedFromDefault: "当前使用插件配置的 default_schedule",
      schedNone: "未设置时段规则：该密钥任何时间都可以使用。",
      schedWindows: "个时段",
      schedHint: "某一时刻按第一个覆盖它的时段处理，因此午休这类较窄的时段应排在较宽的时段之前。结束时间早于或等于开始时间表示跨过午夜；两者相同表示全天。时段限额在每次时段开始时重新计算。",
      schedOutside: "所有时段之外",
      schedAllow: "允许使用，按常规限额",
      schedBlockAll: "禁止使用",
      addWindow: "添加时段",
      winName: "时段名称",
      moveUp: "上移",
      winLimit: "按限额使用",
      winBlock: "禁止使用",
      winQuota: "时段限额",
      winRate: "时段速率",
      winNoQuota: "不设时段限额",
      winNeedDays: "每个时段至少要选择一天。",
      winNeedTime: "每个时段都需要开始和结束时间。",
      dayNames: "一,二,三,四,五,六,日",
      inWindow: "时段",
      outsideSchedule: "时段外",
      blockedWindow: "禁用时段",
      windowUntil: "至",
      showPie: "显示饼图",
      metricCost: "费用",
      metricTokens: "Token",
      metricRequests: "请求",
      otherModels: "其他",
      pieShare: "占比",
      pieTooFew: "至少需要两个有用量的模型才能显示饼图。",
      tabAccounts: "账号",
      accountsTitle: "账号",
      accountsHint: "宿主中的账号凭证：当前正在处理的请求数、适用的规则，以及绑定到它的密钥。",
      accountsListFailed: "宿主未返回账号列表，下面只显示正在处理请求的账号",
      noAccounts: "暂无账号。",
      colAccount: "账号",
      colLive: "并发",
      colRule: "规则",
      colBoundKeys: "绑定的密钥",
      colConcurrency: "并发上限",
      colExclusive: "独占",
      accountRules: "账号规则",
      accountRulesHint: "按账号 ID 匹配，第一条匹配的规则生效。并发上限限制账号同时处理的请求数，留空或 0 表示不限。独占账号只为绑定了它的密钥服务。",
      noAccountRules: "未设置账号规则：所有账号为所有密钥服务，且不限并发。",
      exclusivePill: "独占",
      unavailablePill: "不可用",
      disabledPill: "已禁用",
      unlimited: "不限",
      noRule: "无规则",
      sharedAccount: "共享",
      rateQueue: "排队等待（秒）",
      rateQueueHint: "当密钥或其可用的全部账号达到并发上限时，请求最多等待这么久，仍没有空位才会被拒绝。",
      keyAccounts: "该密钥绑定的账号",
      keyAccountsHint: "填写账号 ID 或通配符。该密钥的请求优先调度到这些账号，留空表示可以使用所有账号。",
      noKeyAccounts: "未绑定：该密钥使用所有共享账号。",
      phAccount: "codex-team-*.json",
      strictAccounts: "只使用这些账号",
      strictAccountsHint: "绑定的账号都无法处理请求时，直接返回失败，不退回到共享账号。",
      boundPill: "账号",
      strictPill: "严格"
    }
  };

  var lang = initialLanguage();
  var keyInput = document.getElementById("mgmtKey");
  var statusEl = document.getElementById("status");
  var rowsEl = document.getElementById("rows");
  var cardsEl = document.getElementById("cards");
  var priceRowsEl = document.getElementById("priceRows");
  var observedRowsEl = document.getElementById("observedRows");
  var blockedEl = document.getElementById("globalBlocked");
  var fallbacksEl = document.getElementById("globalFallbacks");
  var editor = document.getElementById("editor");
  var resetDialog = document.getElementById("resetDialog");
  var resetting = [];
  var deleteDialog = document.getElementById("deleteDialog");
  var deleting = [];
  // selected holds the ids of the rows picked for a batch action. It survives
  // refreshes and filtering, and drops ids that disappear from the usage list.
  var selected = {};
  // shownKeys are the rows the filter currently lets through; select all acts
  // on them only.
  var shownKeys = [];
  var modelsDialog = document.getElementById("modelsDialog");
  var settingsDialog = document.getElementById("settingsDialog");
  var filterInput = document.getElementById("keyFilter");
  var editing = null;
  var keyBlockedPatterns = [];
  var keyMappings = [];
  var mappingRowsEl = document.getElementById("mappingRows");
  // mappingsDirty keeps auto refresh from discarding unsaved rule edits.
  var mappingsDirty = false;
  var timer = null;
  var activeTab = "keys";
  var lastUsage = null;
  var lastPricing = null;
  var lastAccounts = null;
  var keyAccounts = [];
  var accountRowsEl = document.getElementById("accountRows");
  var accountRuleRowsEl = document.getElementById("accountRuleRows");
  // accountRulesDirty keeps auto refresh from discarding unsaved rule edits.
  var accountRulesDirty = false;

  function initialLanguage() {
    try {
      var saved = localStorage.getItem(LANG_STORAGE);
      if (saved === "en" || saved === "zh") { return saved; }
    } catch (err) { /* storage may be unavailable */ }
    var nav = (navigator.language || "en").toLowerCase();
    return nav.indexOf("zh") === 0 ? "zh" : "en";
  }

  function t(key) {
    var table = I18N[lang] || I18N.en;
    if (table[key] !== undefined) { return table[key]; }
    return I18N.en[key] !== undefined ? I18N.en[key] : key;
  }

  function locale() { return lang === "zh" ? "zh-CN" : "en-US"; }

  function applyStatic() {
    var nodes = document.querySelectorAll("[data-i18n]");
    for (var i = 0; i < nodes.length; i++) {
      nodes[i].textContent = t(nodes[i].getAttribute("data-i18n"));
    }
    var placeholders = document.querySelectorAll("[data-i18n-ph]");
    for (var j = 0; j < placeholders.length; j++) {
      placeholders[j].placeholder = t(placeholders[j].getAttribute("data-i18n-ph"));
    }
    var titles = document.querySelectorAll("[data-i18n-title]");
    for (var k = 0; k < titles.length; k++) {
      titles[k].title = t(titles[k].getAttribute("data-i18n-title"));
      titles[k].setAttribute("aria-label", titles[k].title);
    }
    document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
    document.getElementById("langEn").className = lang === "en" ? "on" : "";
    document.getElementById("langZh").className = lang === "zh" ? "on" : "";
    document.getElementById("revealKey").textContent = keyInput.type === "text" ? t("hide") : t("show");
  }

  function setLanguage(next) {
    lang = next;
    try { localStorage.setItem(LANG_STORAGE, next); } catch (err) { /* ignore */ }
    applyStatic();
    if (lastUsage) { renderCards(lastUsage); renderRows(lastUsage); renderBlocked(lastUsage); renderMappings(lastUsage); renderFallbacks(lastUsage); }
    else { showMessage(t("enterKey")); }
    if (lastPricing) { renderPricing(lastPricing); renderObserved(lastPricing); }
    if (lastAccounts) { renderAccounts(lastAccounts); renderAccountRules(lastAccounts); }
  }

  // normalizeKey drops the zero-width characters that survive String.trim() and
  // ride along with a pasted key, which otherwise fail authentication invisibly.
  function normalizeKey() {
    var raw = keyInput.value;
    var out = "";
    for (var i = 0; i < raw.length; i++) {
      var code = raw.charCodeAt(i);
      if (code === 0x200B || code === 0x200C || code === 0x200D || code === 0xFEFF) { continue; }
      out += raw.charAt(i);
    }
    return out.trim();
  }

  function headers() {
    var out = { "Content-Type": "application/json" };
    var value = normalizeKey();
    if (value) { out["X-Management-Key"] = value; }
    return out;
  }

  function persistKey() {
    try { sessionStorage.setItem(KEY_STORAGE, normalizeKey()); } catch (err) { /* ignore */ }
  }

  function forgetKey() {
    try { sessionStorage.removeItem(KEY_STORAGE); } catch (err) { /* ignore */ }
  }

  // requestFailed surfaces the JSON error message the host returns instead of a
  // bare status code, and carries the status so callers can react to auth errors.
  function requestFailed(response) {
    return response.text().then(function (text) {
      var message = "HTTP " + response.status;
      try {
        var parsed = JSON.parse(text);
        if (parsed && parsed.error) { message = parsed.error + " (HTTP " + response.status + ")"; }
      } catch (err) { /* body is not JSON */ }
      var failure = new Error(message);
      failure.status = response.status;
      throw failure;
    });
  }

  // handleFailure stops auto refresh on an auth error. The host bans a client IP
  // for 30 minutes after five rejected management keys, so a tab left open with a
  // wrong key must not keep retrying.
  function handleFailure(err, prefix) {
    if (err.status === 401 || err.status === 403) {
      stopRefresh();
      forgetKey();
      showMessage(t("authFailed"));
      setStatus(prefix + ": " + err.message +
        " - " + t("sent") + " " + normalizeKey().length + " " + t("chars") +
        " - " + t("refreshStopped"), true);
      return;
    }
    setStatus(prefix + ": " + err.message, true);
  }

  // setStatus drives the status line under the title; state is "ok", "bad" or
  // empty for a neutral message such as loading.
  function setStatus(text, isError, state) {
    statusEl.textContent = text;
    document.getElementById("statusLine").className =
      "statusline" + (isError ? " bad" : (state ? " " + state : ""));
  }

  // showMessage replaces the key table with a notice. Without usage data every
  // notice is about the management key, so it offers the settings dialog.
  function showMessage(text) {
    lastUsage = null;
    cardsEl.innerHTML = "";
    rowsEl.innerHTML = "";
    var row = messageRow(text, 10);
    var button = actionButton(t("setKey"), openSettings, "primary");
    button.className = "btn primary";
    row.firstChild.appendChild(el("br"));
    row.firstChild.appendChild(button);
    rowsEl.appendChild(row);
    updateCounts();
  }

  function messageRow(text, span) {
    var row = document.createElement("tr");
    var cell = document.createElement("td");
    cell.colSpan = span;
    cell.className = "empty";
    cell.textContent = text;
    row.appendChild(cell);
    return row;
  }

  function openSettings() {
    settingsDialog.showModal();
    keyInput.focus();
  }

  // updateCounts refreshes the badges on the tabs from the latest responses.
  function updateCounts() {
    var usage = lastUsage;
    document.getElementById("countKeys").textContent = usage ? String(usage.keys.length) : "";
    var rules = usage ? (usage.blocked_models || []).length + (usage.model_mappings || []).length +
      (usage.fallback_models || []).length : 0;
    document.getElementById("countRules").textContent = rules ? String(rules) : "";
    var prices = lastPricing ? (lastPricing.models || []).length : 0;
    document.getElementById("countPricing").textContent = prices ? String(prices) : "";
    var accounts = lastAccounts ? lastAccounts.accounts.length : 0;
    document.getElementById("countAccounts").textContent = accounts ? String(accounts) : "";
    document.getElementById("dirtyRules").hidden = !mappingsDirty;
    document.getElementById("dirtyAccounts").hidden = !accountRulesDirty;
  }

  function setAccountRulesDirty(value) {
    accountRulesDirty = value;
    updateCounts();
  }

  function setMappingsDirty(value) {
    mappingsDirty = value;
    updateCounts();
  }

  // syncTheme copies the host's theme when the page is embedded in the
  // management center. Cross-origin frames throw, and the page then follows the
  // browser preference through the stylesheet.
  function syncTheme() {
    var root;
    try {
      if (window.parent === window) { return; }
      root = window.parent.document.documentElement;
    } catch (err) { return; }
    function apply() {
      var theme = root.getAttribute("data-theme");
      if (theme === "dark" || theme === "white") {
        document.documentElement.setAttribute("data-theme", theme);
      } else {
        // The host removes the attribute for its default warm light theme.
        document.documentElement.setAttribute("data-theme", "light");
      }
    }
    apply();
    try {
      new MutationObserver(apply).observe(root, { attributes: true, attributeFilter: ["data-theme"] });
    } catch (err) { /* observation is best effort */ }
  }

  function el(tag, className, text) {
    var node = document.createElement(tag);
    if (className) { node.className = className; }
    if (text !== undefined && text !== null) { node.textContent = text; }
    return node;
  }

  function num(value) {
    if (value === null || value === undefined) { return t("unlimited"); }
    return Number(value).toLocaleString(locale());
  }

  // tokenNum spells token counts out with thousands separators until they reach
  // the large unit, then counts in it: 亿 (10^8) in Chinese, B (10^9) in English.
  // 1,700,000 stays as is, while 170,000,000,000 reads 1,700亿 or 170B.
  function tokenNum(value) {
    if (value === null || value === undefined) { return t("unlimited"); }
    var parsed = Number(value);
    var unit = lang === "zh" ? 1e8 : 1e9;
    if (Math.abs(parsed) < unit) { return parsed.toLocaleString("en-US"); }
    return (parsed / unit).toLocaleString("en-US", { maximumFractionDigits: 2 }) +
      (lang === "zh" ? "亿" : "B");
  }

  function money(value) { return "$" + Number(value || 0).toFixed(4); }

  function price(value) {
    var parsed = Number(value || 0);
    return parsed === 0 ? "-" : parsed.toLocaleString(locale(), { maximumFractionDigits: 6 });
  }

  function meter(used, limit) {
    var wrap = el("div", "meter");
    var fill = el("span");
    var ratio = limit > 0 ? Math.min(1, used / limit) : 0;
    fill.style.width = (ratio * 100).toFixed(1) + "%";
    if (ratio >= 1) { wrap.className = "meter bad"; }
    else if (ratio >= 0.8) { wrap.className = "meter warn"; }
    wrap.appendChild(fill);
    return wrap;
  }

  function dimension(stack, label, used, limit, format, extra) {
    var line = el("div", "dimline" + (extra ? " " + extra : ""));
    line.appendChild(el("span", "tag", label));
    var value = el("span", "mono", format(used));
    line.appendChild(value);
    if (limit !== null && limit !== undefined) {
      line.appendChild(el("span", "mono faint", "/ " + format(limit)));
    }
    stack.appendChild(line);
    if (limit !== null && limit !== undefined) { stack.appendChild(meter(used, limit)); }
  }

  // rateCell shows the last minute of activity against the rate limits in
  // force, and the window quota when a schedule window is active.
  function rateCell(item) {
    var cell = el("td");
    var stack = el("div", "stack");
    var rate = item.rate || {};
    var concurrency = rate.concurrency || { used: item.concurrent || 0, limit: null };
    dimension(stack, t("tagConc"), concurrency.used, concurrency.limit, num, "rateline");
    if (rate.rpm) { dimension(stack, "RPM", rate.rpm.used, rate.rpm.limit, num, "rateline"); }
    if (rate.tpm) { dimension(stack, "TPM", rate.tpm.used, rate.tpm.limit, tokenNum, "rateline"); }
    if (item.window && !item.window.block) {
      var limits = item.window.limits;
      var used = item.window.used;
      if (limits.cost_usd !== null) { dimension(stack, "W$", used.cost_usd, limits.cost_usd, money, "rateline"); }
      if (limits.tokens !== null) { dimension(stack, "WT", used.tokens, limits.tokens, tokenNum, "rateline"); }
      if (limits.requests !== null) { dimension(stack, "WR", used.requests, limits.requests, num, "rateline"); }
    }
    cell.appendChild(stack);
    return cell;
  }

  function periodCell(period) {
    var cell = el("td");
    var stack = el("div", "stack");
    dimension(stack, "$", period.used.cost_usd, period.limits.cost_usd, money);
    dimension(stack, "T", period.used.tokens, period.limits.tokens, tokenNum);
    dimension(stack, "R", period.used.requests, period.limits.requests, num);
    cell.appendChild(stack);
    return cell;
  }

  function pill(text, cls) {
    return el("span", "pill " + (cls || ""), text);
  }

  function renderCards(data) {
    cardsEl.innerHTML = "";
    var rejected = data.keys.reduce(function (sum, item) { return sum + item.blocked; }, 0);
    var items = [
      [t("cardKeys"), num(data.keys.length)],
      [t("cardActive"), num(data.concurrent || 0), data.concurrent ? "live" : ""],
      [t("cardTokens"), tokenNum(data.totals.tokens)],
      [t("cardRequests"), num(data.totals.requests)],
      [t("cardCost"), money(data.totals.cost_usd)],
      [t("cardBlocked"), num(rejected), rejected ? "bad" : ""]
    ];
    items.forEach(function (item) {
      var card = el("div", "card" + (item[2] ? " " + item[2] : ""));
      card.appendChild(el("div", "label", item[0]));
      card.appendChild(el("div", "value", item[1]));
      cardsEl.appendChild(card);
    });
  }

  function matchesFilter(item, query) {
    if (!query) { return true; }
    return [item.label, item.hint, item.id, item.note].some(function (value) {
      return value && String(value).toLowerCase().indexOf(query) !== -1;
    });
  }

  function renderRows(data) {
    rowsEl.innerHTML = "";
    var info = document.getElementById("keyFilterInfo");
    info.textContent = "";
    var known = {};
    data.keys.forEach(function (item) { known[item.id] = true; });
    Object.keys(selected).forEach(function (id) { if (!known[id]) { delete selected[id]; } });
    shownKeys = [];
    if (!data.keys.length) {
      rowsEl.appendChild(messageRow(t("noUsage"), 10));
      updateSelection();
      return;
    }
    var query = filterInput.value.trim().toLowerCase();
    var shown = data.keys.filter(function (item) { return matchesFilter(item, query); });
    if (query) { info.textContent = shown.length + " / " + data.keys.length + " " + t("filterInfo"); }
    shownKeys = shown;
    if (!shown.length) {
      rowsEl.appendChild(messageRow(t("noMatch"), 10));
      updateSelection();
      return;
    }
    shown.forEach(function (item) {
      var row = document.createElement("tr");
      row.setAttribute("data-id", item.id);

      var pickCell = el("td", "sel");
      var pick = el("input", "pick");
      pick.type = "checkbox";
      pick.checked = !!selected[item.id];
      pick.title = t("selectRow");
      pick.setAttribute("aria-label", pick.title);
      pick.addEventListener("change", function () {
        if (pick.checked) { selected[item.id] = true; } else { delete selected[item.id]; }
        updateSelection();
      });
      pickCell.appendChild(pick);
      row.appendChild(pickCell);

      var identity = el("td");
      var stack = el("div", "stack");
      if (item.note) {
        var noteLine = el("div", "key-note wrapcell", item.note);
        noteLine.title = item.note;
        stack.appendChild(noteLine);
      }
      stack.appendChild(el("div", "key-name", item.label || item.hint || item.id));
      stack.appendChild(el("div", "mono faint", item.id));
      var badges = el("div", "badges");
      if (item.concurrent) {
        row.className += " live";
        badges.appendChild(pill(t("inUse"), "live"));
      }
      if (item.disabled) {
        row.className += " paused";
        badges.appendChild(pill(t("paused"), "bad"));
      }
      if (item.outside_schedule) {
        badges.appendChild(pill(item.window ? t("blockedWindow") + ": " + item.window.name : t("outsideSchedule"), "bad"));
      } else if (item.window) {
        var windowPill = pill(t("inWindow") + ": " + item.window.name, "on");
        windowPill.title = t("windowUntil") + " " + new Date(item.window.end).toLocaleString(locale());
        badges.appendChild(windowPill);
      }
      if (item.overridden) { badges.appendChild(pill(t("overridden"), "on")); }
      if (item.mapped) { badges.appendChild(pill(t("mappedPill") + " " + num(item.mapped))); }
      if (item.fallbacks) { badges.appendChild(pill(t("fallbackPill") + " " + num(item.fallbacks))); }
      if (item.accounts && item.accounts.length) {
        var boundPill = pill(t("boundPill") + " " + item.accounts.length + (item.strict_accounts ? " · " + t("strictPill") : ""), "on");
        boundPill.title = item.accounts.join("\n");
        badges.appendChild(boundPill);
      }
      if (badges.children.length) { stack.appendChild(badges); }
      identity.appendChild(stack);
      row.appendChild(identity);

      row.appendChild(rateCell(item));

      row.appendChild(periodCell(item.periods.daily));
      row.appendChild(periodCell(item.periods.monthly));
      row.appendChild(periodCell(item.periods.total));

      var models = el("td", "wrapcell mono hint");
      models.textContent = item.blocked_models.length ? item.blocked_models.join(", ") : t("none");
      row.appendChild(models);

      row.appendChild(el("td", "num", num(item.blocked)));
      row.appendChild(el("td", "hint", item.last_used_at || "-"));

      // Everyday actions first; the destructive reset sits last and reads as such.
      var actions = el("td", "actions");
      var group = el("div", "rowbtns");
      group.appendChild(actionButton(t("limitsBtn"), function () { openEditor(item); }));
      group.appendChild(actionButton(t("modelsBtn"), function () { openModels(item); }));
      group.appendChild(actionButton(item.disabled ? t("enableBtn") : t("pauseBtn"), function () {
        togglePause(item);
      }, "ghost"));
      group.appendChild(actionButton(t("rotateBtn"), function () { openRotate([item]); }, "ghost"));
      group.appendChild(actionButton(t("resetBtn"), function () { resetKey([item]); }, "danger"));
      group.appendChild(actionButton(t("deleteBtn"), function () { openDelete([item]); }, "danger"));
      actions.appendChild(group);
      row.appendChild(actions);

      rowsEl.appendChild(row);
    });
    updateSelection();
  }

  function selectedItems() {
    if (!lastUsage) { return []; }
    return lastUsage.keys.filter(function (item) { return selected[item.id]; });
  }

  // updateSelection syncs the batch bar, the select-all box and the row
  // highlight with the current selection.
  function updateSelection() {
    var count = selectedItems().length;
    document.getElementById("selBar").hidden = count === 0;
    document.getElementById("selCount").textContent = t("selectedCount") + " " + count;
    var all = document.getElementById("selectAll");
    var picked = shownKeys.filter(function (item) { return selected[item.id]; }).length;
    all.checked = shownKeys.length > 0 && picked === shownKeys.length;
    all.indeterminate = picked > 0 && picked < shownKeys.length;
    all.disabled = shownKeys.length === 0;
    var rows = rowsEl.querySelectorAll("tr[data-id]");
    for (var i = 0; i < rows.length; i++) {
      rows[i].classList.toggle("picked", !!selected[rows[i].getAttribute("data-id")]);
    }
  }

  function clearSelection() {
    selected = {};
    if (lastUsage) { renderRows(lastUsage); } else { updateSelection(); }
  }

  function displayName(item) {
    return item.note || item.label || item.hint || item.id;
  }

  // dialogTitle names the single key a dialog acts on, or counts the batch.
  function dialogTitle(prefix, items) {
    return prefix + ": " + (items.length === 1 ? displayName(items[0]) : items.length + " " + t("keysUnit"));
  }

  // fillTargets lists the keys a dialog acts on. Unless always is set, the
  // list only shows for a batch, since the title already names a single key.
  function fillTargets(listId, items, always) {
    var list = document.getElementById(listId);
    list.innerHTML = "";
    list.hidden = !always && items.length < 2;
    items.forEach(function (item) {
      var line = el("li");
      line.appendChild(el("span", null, displayName(item)));
      line.appendChild(el("span", "mono", item.hint && item.hint !== displayName(item) ? item.hint : item.id));
      list.appendChild(line);
    });
  }

  function batchPause(disabled) {
    var ids = selectedItems().map(function (item) { return item.id; });
    if (!ids.length) { return; }
    post("/limits", { ids: ids, disabled: disabled })
      .then(function () { setStatus(t("batchSaved"), false, "ok"); load(); })
      .catch(function (err) { handleFailure(err, t("saveFailed")); });
  }

  // actionButton builds a compact button; variant is "primary", "ghost",
  // "danger" or empty for the neutral style.
  function actionButton(text, handler, variant) {
    var button = el("button", "btn sm" + (variant ? " " + variant : ""), text);
    button.type = "button";
    button.addEventListener("click", handler);
    return button;
  }

  function renderBlocked(data) {
    blockedEl.value = (data.blocked_models || []).join("\n");
    document.getElementById("blockedState").textContent =
      data.blocked_models_overridden ? t("customValues") : t("fromConfig");
    document.getElementById("blockedState").className = data.blocked_models_overridden ? "pill on" : "pill";
  }

  function renderFallbacks(data) {
    fallbacksEl.value = (data.fallback_models || []).join("\n");
    var state = document.getElementById("fallbacksState");
    state.textContent = data.fallback_models_overridden ? t("customValues") : t("fromConfig");
    state.className = data.fallback_models_overridden ? "pill on" : "pill";
  }

  // syncFallbacks shows the key's own list, or names the global one it follows.
  function syncFallbacks() {
    var own = document.getElementById("fallbackOwn").checked;
    document.getElementById("keyFallbacks").hidden = !own;
    var source = document.getElementById("fallbackSource");
    if (own) { source.textContent = t("fallbackOwnHint"); return; }
    var global = lastUsage ? lastUsage.fallback_models || [] : [];
    source.textContent = global.length ? t("fallbackInherited") + ": " + global.join(" → ") : t("fallbackNone");
  }

  function fillFallbacks(item) {
    var own = item.own_fallback_models;
    document.getElementById("fallbackOwn").checked = own != null;
    // The list in force seeds the editor when the operator opts in.
    document.getElementById("keyFallbacks").value = (own != null ? own : item.fallback_models || []).join("\n");
    syncFallbacks();
  }

  function mappingText(rule) {
    return rule.from + " \u2192 " + rule.to + (rule.provider ? " @" + rule.provider : "");
  }

  function textInput(value, placeholder, listId, width) {
    var input = el("input");
    input.type = "text";
    input.className = "mono";
    input.spellcheck = false;
    input.autocomplete = "off";
    input.style.minWidth = width;
    input.value = value || "";
    input.placeholder = placeholder;
    if (listId) { input.setAttribute("list", listId); }
    return input;
  }

  function addMappingRow(rule) {
    if (mappingRowsEl.children.length === 1 && !mappingRowsEl.querySelectorAll("input").length) {
      mappingRowsEl.innerHTML = "";
    }
    var row = document.createElement("tr");
    [
      textInput(rule && rule.from, "gpt-4o*", "modelSuggestions", "170px"),
      textInput(rule && rule.to, "gpt-5-mini", "modelSuggestions", "170px"),
      textInput(rule && rule.provider, t("autoProvider"), "providerSuggestions", "110px")
    ].forEach(function (input) {
      var cell = el("td");
      cell.appendChild(input);
      row.appendChild(cell);
    });
    var actions = el("td");
    actions.appendChild(actionButton(t("remove"), function () {
      row.parentNode.removeChild(row);
      if (!mappingRowsEl.children.length) { mappingRowsEl.appendChild(messageRow(t("noMappings"), 4)); }
    }, "danger"));
    row.appendChild(actions);
    mappingRowsEl.appendChild(row);
  }

  function fillProviders(providers) {
    var list = document.getElementById("providerSuggestions");
    list.innerHTML = "";
    (providers || []).forEach(function (value) {
      var option = document.createElement("option");
      option.value = value;
      list.appendChild(option);
    });
    document.getElementById("providersHint").textContent = providers && providers.length ?
      t("providersSeen") + ": " + providers.join(", ") : t("providersUnknown");
  }

  function renderMappings(data) {
    fillProviders(data.providers);
    if (mappingsDirty) { return; }
    mappingRowsEl.innerHTML = "";
    (data.model_mappings || []).forEach(function (rule) { addMappingRow(rule); });
    if (!mappingRowsEl.children.length) { mappingRowsEl.appendChild(messageRow(t("noMappings"), 4)); }
    document.getElementById("mappingsState").textContent =
      data.model_mappings_overridden ? t("customValues") : t("fromConfig");
    document.getElementById("mappingsState").className = data.model_mappings_overridden ? "pill on" : "pill";
  }

  function collectMappings() {
    var out = [];
    var rows = mappingRowsEl.querySelectorAll("tr");
    for (var i = 0; i < rows.length; i++) {
      var inputs = rows[i].querySelectorAll("input");
      if (inputs.length < 3) { continue; }
      var rule = { from: inputs[0].value.trim(), to: inputs[1].value.trim(), provider: inputs[2].value.trim() };
      if (!rule.from && !rule.to && !rule.provider) { continue; }
      out.push(rule);
    }
    return out;
  }

  function renderKeyMappings() {
    var list = document.getElementById("keyMappingList");
    list.innerHTML = "";
    if (!keyMappings.length) {
      list.appendChild(el("li", "empty", t("noKeyMappings")));
      return;
    }
    keyMappings.forEach(function (rule, index) {
      var item = el("li", null);
      item.appendChild(el("span", null, mappingText(rule)));
      var remove = el("button", null, "\u00D7");
      remove.type = "button";
      remove.title = t("removeTag");
      remove.addEventListener("click", function () {
        keyMappings.splice(index, 1);
        renderKeyMappings();
      });
      item.appendChild(remove);
      list.appendChild(item);
    });
  }

  var CUSTOM_FROM = "__custom__";

  // mapFromValue reads the requested model from the dropdown, or from the free
  // text field when the operator picked a custom pattern.
  function mapFromValue() {
    var select = document.getElementById("keyMapFromSelect");
    if (select.value === CUSTOM_FROM) { return document.getElementById("keyMapFrom").value.trim(); }
    return select.value;
  }

  function syncMapFrom() {
    var custom = document.getElementById("keyMapFromSelect").value === CUSTOM_FROM;
    var input = document.getElementById("keyMapFrom");
    input.hidden = !custom;
    if (custom) { input.focus(); } else { input.value = ""; }
  }

  function resetMapFrom() {
    document.getElementById("keyMapFromSelect").value = "";
    syncMapFrom();
  }

  function addKeyMapping() {
    var from = mapFromValue();
    var to = document.getElementById("keyMapTo");
    var provider = document.getElementById("keyMapProvider");
    if (!from || !to.value.trim()) {
      document.getElementById("editError").textContent = t("mapIncomplete");
      return;
    }
    document.getElementById("editError").textContent = "";
    var lowered = from.toLowerCase();
    keyMappings = keyMappings.filter(function (rule) { return rule.from.toLowerCase() !== lowered; });
    keyMappings.push({ from: from, to: to.value.trim(), provider: provider.value.trim() });
    resetMapFrom();
    to.value = "";
    provider.value = "";
    renderKeyMappings();
  }

  function priceInput(value) {
    var input = el("input");
    input.type = "number";
    input.step = "any";
    input.style.width = "96px";
    input.value = value ? String(value) : "";
    return input;
  }

  function addPriceRow(entry) {
    var row = document.createElement("tr");
    var matchCell = el("td");
    var match = el("input");
    match.type = "text";
    match.className = "mono";
    match.style.minWidth = "160px";
    match.value = entry && entry.match ? entry.match : "";
    match.placeholder = "gpt-5*";
    matchCell.appendChild(match);
    row.appendChild(matchCell);

    var fields = ["input", "output", "reasoning", "cache_read", "cache_write"];
    fields.forEach(function (field) {
      var cell = el("td", "num");
      var input = priceInput(entry ? entry[field] : 0);
      cell.appendChild(input);
      row.appendChild(cell);
    });

    var actions = el("td");
    actions.appendChild(actionButton(t("remove"), function () { row.parentNode.removeChild(row); }, "danger"));
    row.appendChild(actions);

    priceRowsEl.appendChild(row);
    return row;
  }

  function renderPricing(data) {
    document.getElementById("p_def_input").value = data.default.input || "";
    document.getElementById("p_def_output").value = data.default.output || "";
    document.getElementById("p_def_reasoning").value = data.default.reasoning || "";
    document.getElementById("p_def_cache_read").value = data.default.cache_read || "";
    document.getElementById("p_def_cache_write").value = data.default.cache_write || "";

    priceRowsEl.innerHTML = "";
    (data.models || []).forEach(function (entry) { addPriceRow(entry); });
    if (!priceRowsEl.children.length) { priceRowsEl.appendChild(messageRow(t("noPrices"), 7)); }
    var source = t("srcConfig");
    if (data.source === "builtin") { source = t("srcBuiltin"); }
    else if (data.source === "runtime") { source = t("srcRuntime"); }
    document.getElementById("priceSource").textContent = source;
    document.getElementById("pricingState").textContent =
      data.overridden ? t("customValues") : t("fromConfig");
    document.getElementById("pricingState").className = data.overridden ? "pill on" : "pill";
    updateCounts();
  }

  var PIE_STORAGE = "apikey-quota-observed-pie";

  function observedPieOn() {
    try { return localStorage.getItem(PIE_STORAGE) === "1"; } catch (err) { return false; }
  }

  // renderObservedPie charts every recognized model when the switch is on,
  // ranked by spend, then tokens, so slice colours stay with their models.
  function renderObservedPie(data) {
    var viz = document.getElementById("observedViz");
    var on = document.getElementById("observedPie").checked;
    viz.hidden = !on;
    if (!on) { return; }
    var observed = (data.observed || []).filter(function (item) {
      return item.used.tokens || item.used.requests || item.used.cost_usd;
    });
    if (observed.length < 2) {
      viz.innerHTML = "";
      viz.appendChild(el("p", "viz-empty", t("pieTooFew")));
      return;
    }
    var priced = observed.some(function (item) { return item.used.cost_usd > 0; });
    observed.sort(function (a, b) {
      if (b.used.cost_usd !== a.used.cost_usd) { return b.used.cost_usd - a.used.cost_usd; }
      return b.used.tokens - a.used.tokens;
    });
    renderPie(viz, observed.map(function (item) { return { name: item.model, used: item.used }; }),
      priced ? "cost_usd" : "tokens");
  }

  function renderObserved(data) {
    renderObservedPie(data);
    observedRowsEl.innerHTML = "";
    var observed = data.observed || [];
    if (!observed.length) {
      observedRowsEl.appendChild(messageRow(t("noModels"), 8));
      return;
    }
    observed.forEach(function (item) {
      var row = document.createElement("tr");
      row.appendChild(el("td", "mono", item.model));
      row.appendChild(el("td", "mono hint", item.matched || t("defaultRule")));
      row.appendChild(el("td", "num mono", price(item.price.input)));
      row.appendChild(el("td", "num mono", price(item.price.output)));
      row.appendChild(el("td", "num", tokenNum(item.used.tokens)));
      row.appendChild(el("td", "num", num(item.used.requests)));
      row.appendChild(el("td", "num", money(item.used.cost_usd)));
      var flag = el("td");
      if (item.blocked) { flag.appendChild(pill(t("blocked"), "bad")); }
      if (!item.priced) { flag.appendChild(pill(t("unpriced"), "warn")); }
      row.appendChild(flag);
      observedRowsEl.appendChild(row);
    });
  }

  function collectPricing() {
    var payload = {
      "default": {
        input: numberOf("p_def_input"),
        output: numberOf("p_def_output"),
        reasoning: numberOf("p_def_reasoning"),
        cache_read: numberOf("p_def_cache_read"),
        cache_write: numberOf("p_def_cache_write")
      },
      models: []
    };
    var rows = priceRowsEl.querySelectorAll("tr");
    for (var i = 0; i < rows.length; i++) {
      var inputs = rows[i].querySelectorAll("input");
      if (inputs.length < 6) { continue; }
      var match = inputs[0].value.trim();
      if (!match) { continue; }
      payload.models.push({
        match: match,
        input: valueOf(inputs[1]),
        output: valueOf(inputs[2]),
        reasoning: valueOf(inputs[3]),
        cache_read: valueOf(inputs[4]),
        cache_write: valueOf(inputs[5])
      });
    }
    return payload;
  }

  function valueOf(input) {
    var raw = input.value.trim();
    if (!raw) { return 0; }
    var parsed = Number(raw);
    return isNaN(parsed) ? 0 : parsed;
  }

  function numberOf(id) { return valueOf(document.getElementById(id)); }

  function lines(value) {
    return value.split("\n").map(function (item) { return item.trim(); })
      .filter(function (item) { return item.length > 0; });
  }

  function load() {
    if (!normalizeKey()) {
      stopRefresh();
      showMessage(t("enterKey"));
      setStatus(t("keyRequired"), true);
      return;
    }
    setStatus(t("loading"));
    var reload = document.getElementById("reload");
    reload.classList.add("spin");
    fetch(base + "/usage", { headers: headers(), cache: "no-store" })
      .then(function (response) {
        if (!response.ok) { return requestFailed(response); }
        return response.json();
      })
      .then(function (data) {
        persistKey();
        lastUsage = data;
        renderCards(data);
        renderRows(data);
        renderBlocked(data);
        renderMappings(data);
        renderFallbacks(data);
        updateCounts();
        setStatus(t("updated") + " " + new Date().toLocaleTimeString(locale()) +
          " · " + (data.enforce ? t("enforceOn") : t("enforceOff")) +
          " · " + data.time_zone, false, data.enforce ? "ok" : "");
        scheduleRefresh();
        if (activeTab === "pricing") { loadPricing(); }
        if (activeTab === "accounts") { loadAccounts(); }
      })
      .catch(function (err) {
        handleFailure(err, t("failed"));
      })
      .then(function () { reload.classList.remove("spin"); });
  }

  function loadPricing() {
    if (!normalizeKey()) { return Promise.resolve(); }
    return fetch(base + "/pricing", { headers: headers(), cache: "no-store" })
      .then(function (response) {
        if (!response.ok) { return requestFailed(response); }
        return response.json();
      })
      .then(function (data) {
        lastPricing = data;
        renderPricing(data);
        renderObserved(data);
      })
      .catch(function (err) { handleFailure(err, t("failed")); });
  }

  function loadAccounts() {
    if (!normalizeKey()) { return Promise.resolve(); }
    return fetch(base + "/accounts", { headers: headers(), cache: "no-store" })
      .then(function (response) {
        if (!response.ok) { return requestFailed(response); }
        return response.json();
      })
      .then(renderAccountsData)
      .catch(function (err) { handleFailure(err, t("failed")); });
  }

  function renderAccountsData(data) {
    lastAccounts = data;
    renderAccounts(data);
    renderAccountRules(data);
    fillAccountSuggestions(data);
    updateCounts();
  }

  function accountName(account) {
    return account.label || account.email || account.name || account.id;
  }

  function renderAccounts(data) {
    var listError = document.getElementById("accountsListError");
    listError.hidden = data.listed;
    listError.textContent = data.listed ? "" : t("accountsListFailed") + (data.list_error ? ": " + data.list_error : "");
    accountRowsEl.innerHTML = "";
    if (!data.accounts.length) {
      accountRowsEl.appendChild(messageRow(t("noAccounts"), 4));
      return;
    }
    data.accounts.forEach(function (account) {
      var row = el("tr");
      if (account.concurrent) { row.className = "live"; }
      var identity = el("td");
      var stack = el("div", "stack");
      stack.appendChild(el("div", "key-name", accountName(account)));
      if (accountName(account) !== account.id) { stack.appendChild(el("div", "mono faint", account.id)); }
      var badges = el("div", "badges");
      if (account.provider) { badges.appendChild(pill(account.provider)); }
      if (account.exclusive) { badges.appendChild(pill(t("exclusivePill"), "warn")); }
      if (account.disabled) { badges.appendChild(pill(t("disabledPill"), "bad")); }
      else if (account.unavailable) { badges.appendChild(pill(t("unavailablePill"), "bad")); }
      if (badges.children.length) { stack.appendChild(badges); }
      identity.appendChild(stack);
      row.appendChild(identity);

      var live = el("td");
      var liveStack = el("div", "stack");
      dimension(liveStack, t("tagConc"), account.concurrent, account.limit, num, "rateline");
      live.appendChild(liveStack);
      row.appendChild(live);

      row.appendChild(el("td", "mono hint", account.rule || t("noRule")));
      var keys = el("td", "wrapcell hint");
      keys.textContent = account.bound_keys.length ? account.bound_keys.join(", ") : t("sharedAccount");
      row.appendChild(keys);
      accountRowsEl.appendChild(row);
    });
  }

  function addAccountRuleRow(rule) {
    if (accountRuleRowsEl.children.length === 1 && !accountRuleRowsEl.querySelectorAll("input").length) {
      accountRuleRowsEl.innerHTML = "";
    }
    var row = document.createElement("tr");
    var match = textInput(rule && rule.match, "codex-team-*.json", "accountSuggestions", "220px");
    var concurrency = el("input");
    concurrency.type = "number";
    concurrency.step = "1";
    concurrency.min = "0";
    concurrency.style.width = "110px";
    concurrency.placeholder = t("unlimited");
    concurrency.value = rule && rule.concurrency ? rule.concurrency : "";
    var exclusiveLabel = el("label", "switch");
    var exclusive = el("input");
    exclusive.type = "checkbox";
    exclusive.checked = !!(rule && rule.exclusive);
    exclusiveLabel.appendChild(exclusive);
    exclusiveLabel.appendChild(el("span", "track"));
    [match, concurrency, exclusiveLabel].forEach(function (node) {
      var cell = el("td");
      cell.appendChild(node);
      row.appendChild(cell);
    });
    [match, concurrency, exclusive].forEach(function (input) {
      input.addEventListener("input", function () { setAccountRulesDirty(true); });
      input.addEventListener("change", function () { setAccountRulesDirty(true); });
    });
    var actions = el("td");
    actions.appendChild(actionButton(t("remove"), function () {
      row.parentNode.removeChild(row);
      setAccountRulesDirty(true);
      if (!accountRuleRowsEl.children.length) { accountRuleRowsEl.appendChild(messageRow(t("noAccountRules"), 4)); }
    }, "danger"));
    row.appendChild(actions);
    accountRuleRowsEl.appendChild(row);
  }

  function renderAccountRules(data) {
    var state = document.getElementById("accountRulesState");
    state.textContent = data.rules_overridden ? t("customValues") : t("fromConfig");
    state.className = data.rules_overridden ? "pill on" : "pill";
    if (accountRulesDirty) { return; }
    accountRuleRowsEl.innerHTML = "";
    (data.rules || []).forEach(function (rule) { addAccountRuleRow(rule); });
    if (!accountRuleRowsEl.children.length) { accountRuleRowsEl.appendChild(messageRow(t("noAccountRules"), 4)); }
  }

  function collectAccountRules() {
    var out = [];
    Array.prototype.forEach.call(accountRuleRowsEl.children, function (row) {
      var inputs = row.querySelectorAll("input");
      if (inputs.length < 3) { return; }
      var match = inputs[0].value.trim();
      if (!match) { return; }
      out.push({
        match: match,
        concurrency: Math.max(0, Math.trunc(valueOf(inputs[1]))),
        exclusive: inputs[2].checked
      });
    });
    return out;
  }

  function fillAccountSuggestions(data) {
    var list = document.getElementById("accountSuggestions");
    list.innerHTML = "";
    (data.accounts || []).forEach(function (account) {
      var option = document.createElement("option");
      option.value = account.id;
      if (accountName(account) !== account.id) { option.label = accountName(account); }
      list.appendChild(option);
    });
  }

  function renderKeyAccounts() {
    var list = document.getElementById("keyAccountList");
    list.innerHTML = "";
    if (!keyAccounts.length) {
      list.appendChild(el("li", "empty", t("noKeyAccounts")));
      return;
    }
    keyAccounts.forEach(function (pattern, index) {
      var item = el("li", null);
      item.appendChild(el("span", null, pattern));
      var remove = el("button", null, "×");
      remove.type = "button";
      remove.title = t("removeTag");
      remove.addEventListener("click", function () {
        keyAccounts.splice(index, 1);
        renderKeyAccounts();
      });
      item.appendChild(remove);
      list.appendChild(item);
    });
  }

  function addKeyAccount() {
    var input = document.getElementById("keyAccountInput");
    var value = input.value.trim();
    if (!value) { return; }
    var lowered = value.toLowerCase();
    var exists = keyAccounts.some(function (pattern) { return pattern.toLowerCase() === lowered; });
    if (!exists) { keyAccounts.push(value); }
    input.value = "";
    renderKeyAccounts();
  }

  function post(path, payload) {
    return fetch(base + path, {
      method: "POST",
      headers: headers(),
      body: JSON.stringify(payload)
    }).then(function (response) {
      if (!response.ok) { return requestFailed(response); }
      return response.json();
    });
  }

  // togglePause flips only the disabled flag, so a paused key keeps its limits
  // and deny list and resumes with them intact.
  function togglePause(item) {
    post("/limits", { id: item.id, disabled: !item.disabled })
      .then(load)
      .catch(function (err) { handleFailure(err, t("saveFailed")); });
  }

  function resetKey(items) {
    if (!items.length) { return; }
    resetting = items;
    document.getElementById("resetTitle").textContent = dialogTitle(t("resetBtn"), items);
    fillTargets("resetTargets", items, false);
    document.getElementById("resetError").textContent = "";
    document.getElementById("resetSelect").value = "daily";
    resetDialog.showModal();
  }

  // A period carries one ceiling at a time. Cost wins when several are set, so
  // reopening the editor shows the dimension an operator is most likely managing.
  function fillLimit(prefix, limit) {
    var dim = "";
    var value = "";
    if (limit) {
      if (limit.cost_usd) { dim = "cost_usd"; value = limit.cost_usd; }
      else if (limit.tokens) { dim = "tokens"; value = limit.tokens; }
      else if (limit.requests) { dim = "requests"; value = limit.requests; }
    }
    document.getElementById(prefix + "_dim").value = dim;
    document.getElementById(prefix + "_value").value = dim ? value : "";
    syncLimitInput(prefix);
  }

  function syncLimitInput(prefix) {
    var dim = document.getElementById(prefix + "_dim").value;
    var input = document.getElementById(prefix + "_value");
    input.disabled = !dim;
    if (!dim) { input.value = ""; }
    input.step = dim === "cost_usd" ? "any" : "1";
  }

  function readLimit(prefix) {
    var limit = { tokens: 0, requests: 0, cost_usd: 0 };
    var dim = document.getElementById(prefix + "_dim").value;
    if (dim) { limit[dim] = numberOf(prefix + "_value"); }
    return limit;
  }

  function openModels(item) {
    document.getElementById("modelsTitle").textContent =
      t("modelsTitle") + ": " + (item.label || item.hint || item.id);
    var body = document.getElementById("modelsRows");
    body.innerHTML = "";
    var viz = document.getElementById("modelsViz");
    viz.hidden = true;

    var models = item.models || {};
    var names = Object.keys(models);
    if (!names.length) {
      body.appendChild(messageRow(t("noModelUsage"), 5));
      modelsDialog.showModal();
      return;
    }

    var total = { tokens: 0, requests: 0, cost_usd: 0 };
    names.forEach(function (name) {
      total.tokens += models[name].tokens;
      total.requests += models[name].requests;
      total.cost_usd += models[name].cost_usd;
    });

    // Rank by spend when anything is priced, otherwise by tokens, so the row
    // order stays meaningful for models the price table does not cover.
    var byCost = total.cost_usd > 0;
    names.sort(function (a, b) {
      if (byCost && models[b].cost_usd !== models[a].cost_usd) {
        return models[b].cost_usd - models[a].cost_usd;
      }
      return models[b].tokens - models[a].tokens;
    });

    names.forEach(function (name) {
      var used = models[name];
      var basis = byCost ? total.cost_usd : total.tokens;
      var value = byCost ? used.cost_usd : used.tokens;
      var row = document.createElement("tr");
      row.appendChild(el("td", "mono", name));
      row.appendChild(el("td", "num", num(used.requests)));
      row.appendChild(el("td", "num", tokenNum(used.tokens)));
      row.appendChild(el("td", "num", money(used.cost_usd)));
      var share = el("td", "num");
      share.appendChild(el("div", null, basis > 0 ? (value / basis * 100).toFixed(1) + "%" : "-"));
      if (basis > 0) {
        var bar = meter(value, basis);
        bar.className = "meter";
        share.appendChild(bar);
      }
      row.appendChild(share);
      body.appendChild(row);
    });

    var totals = document.createElement("tr");
    var label = el("td", null, t("totalRow"));
    label.style.fontWeight = "600";
    totals.appendChild(label);
    totals.appendChild(el("td", "num", num(total.requests)));
    totals.appendChild(el("td", "num", tokenNum(total.tokens)));
    totals.appendChild(el("td", "num", money(total.cost_usd)));
    totals.appendChild(el("td", "num", "100%"));
    body.appendChild(totals);

    if (names.length >= 2) {
      renderPie(viz, names.map(function (name) { return { name: name, used: models[name] }; }),
        byCost ? "cost_usd" : "tokens");
    }
    modelsDialog.showModal();
  }

  function renderKeyBlocked() {
    var list = document.getElementById("keyBlockedList");
    list.innerHTML = "";
    if (!keyBlockedPatterns.length) {
      list.appendChild(el("li", "empty", t("noBlocked")));
      return;
    }
    keyBlockedPatterns.forEach(function (pattern, index) {
      var item = el("li", null);
      item.appendChild(el("span", null, pattern));
      var remove = el("button", null, "\u00D7");
      remove.type = "button";
      remove.title = t("removeTag");
      remove.addEventListener("click", function () {
        keyBlockedPatterns.splice(index, 1);
        renderKeyBlocked();
      });
      item.appendChild(remove);
      list.appendChild(item);
    });
  }

  function addKeyBlocked() {
    var input = document.getElementById("keyBlockedInput");
    var value = input.value.trim();
    if (!value) { return; }
    var lowered = value.toLowerCase();
    var exists = keyBlockedPatterns.some(function (pattern) {
      return pattern.toLowerCase() === lowered;
    });
    if (!exists) { keyBlockedPatterns.push(value); }
    input.value = "";
    renderKeyBlocked();
  }

  // Suggestions come from what the plugin already knows: the models this key has
  // called and the patterns in the price table. The host exposes no model
  // registry to plugins, so the field stays free text.
  function fillSuggestions(item) {
    var seen = {};
    var options = [];
    function add(value) {
      if (!value || seen[value]) { return; }
      seen[value] = true;
      options.push(value);
    }
    Object.keys(item.models || {}).forEach(add);
    if (lastPricing) {
      (lastPricing.observed || []).forEach(function (entry) { add(entry.model); });
      (lastPricing.models || []).forEach(function (entry) { add(entry.match); });
    }
    var list = document.getElementById("modelSuggestions");
    list.innerHTML = "";
    options.forEach(function (value) {
      var option = document.createElement("option");
      option.value = value;
      list.appendChild(option);
    });

    // The requested-model dropdown offers the same models, plus a custom entry
    // for glob patterns. A pending selection survives a refill.
    var select = document.getElementById("keyMapFromSelect");
    var previous = select.value;
    select.innerHTML = "";
    var placeholder = el("option", null, t("pickModel"));
    placeholder.value = "";
    select.appendChild(placeholder);
    options.forEach(function (value) {
      var option = el("option", null, value);
      option.value = value;
      select.appendChild(option);
    });
    var custom = el("option", null, t("customPattern"));
    custom.value = CUSTOM_FROM;
    select.appendChild(custom);
    select.value = options.indexOf(previous) !== -1 || previous === CUSTOM_FROM ? previous : "";
  }

  var PIE_METRICS = [
    ["cost_usd", "metricCost", money],
    ["tokens", "metricTokens", tokenNum],
    ["requests", "metricRequests", num]
  ];
  var SVG_NS = "http://www.w3.org/2000/svg";
  var vizTip = document.getElementById("vizTip");

  function svgEl(tag, attrs) {
    var node = document.createElementNS(SVG_NS, tag);
    Object.keys(attrs || {}).forEach(function (key) { node.setAttribute(key, attrs[key]); });
    return node;
  }

  function metricFormat(metric) {
    for (var i = 0; i < PIE_METRICS.length; i++) {
      if (PIE_METRICS[i][0] === metric) { return PIE_METRICS[i][2]; }
    }
    return num;
  }

  // arcPath draws one donut segment between two angles, in radians from 12 o'clock.
  function arcPath(from, to, outer, inner) {
    var c = 90;
    function point(radius, angle) {
      return (c + radius * Math.sin(angle)).toFixed(3) + " " + (c - radius * Math.cos(angle)).toFixed(3);
    }
    var large = to - from > Math.PI ? 1 : 0;
    return "M " + point(outer, from) + " A " + outer + " " + outer + " 0 " + large + " 1 " + point(outer, to) +
      " L " + point(inner, to) + " A " + inner + " " + inner + " 0 " + large + " 0 " + point(inner, from) + " Z";
  }

  function showTip(event, slice, total, format) {
    vizTip.innerHTML = "";
    vizTip.appendChild(el("div", "mono", slice.name));
    vizTip.appendChild(el("div", null, format(slice.value) + " · " +
      (total > 0 ? (slice.value / total * 100).toFixed(1) : "0") + "%"));
    vizTip.hidden = false;
    var x = event.clientX + 14;
    var y = event.clientY + 14;
    var width = vizTip.offsetWidth;
    if (x + width > window.innerWidth - 8) { x = event.clientX - width - 14; }
    vizTip.style.left = x + "px";
    vizTip.style.top = y + "px";
  }

  function hideTip() { vizTip.hidden = true; }

  // renderPie draws a donut of entries[].used[metric]. Entries arrive in a
  // stable rank that decides their colour, so switching the metric never
  // repaints a model; past seven, the rest fold into one "Other" slice.
  function renderPie(box, entries, metric) {
    box.innerHTML = "";
    box.hidden = false;
    box.classList.remove("focus");
    var format = metricFormat(metric);
    var slices = entries.slice(0, 7).map(function (entry, index) {
      return { name: entry.name, value: Number(entry.used[metric] || 0), color: "var(--series-" + (index + 1) + ")" };
    });
    var rest = entries.slice(7);
    if (rest.length) {
      slices.push({
        name: t("otherModels") + " (" + rest.length + ")",
        value: rest.reduce(function (sum, entry) { return sum + Number(entry.used[metric] || 0); }, 0),
        color: "var(--series-other)"
      });
    }
    var total = slices.reduce(function (sum, slice) { return sum + slice.value; }, 0);

    var svg = svgEl("svg", { viewBox: "0 0 180 180", role: "img" });
    var legend = el("ul", "legend");
    var paths = [];
    var angle = 0;
    slices.forEach(function (slice, index) {
      var share = total > 0 ? slice.value / total : 0;
      var item = el("li");
      var swatch = el("span", "sw");
      swatch.style.background = slice.color;
      item.appendChild(swatch);
      var name = el("span", "name", slice.name);
      name.title = slice.name;
      item.appendChild(name);
      item.appendChild(el("span", "val", format(slice.value)));
      item.appendChild(el("span", "pct", (share * 100).toFixed(1) + "%"));
      legend.appendChild(item);
      var path = null;
      if (share > 0) {
        // A lone full slice is drawn as two halves so the arc stays defined.
        var d = share >= 0.9999 ? arcPath(0, Math.PI, 86, 56) + " " + arcPath(Math.PI, 2 * Math.PI, 86, 56) :
          arcPath(angle, angle + share * 2 * Math.PI, 86, 56);
        path = svgEl("path", { d: d });
        path.style.fill = slice.color;
        svg.appendChild(path);
        angle += share * 2 * Math.PI;
      }
      paths[index] = path;
      function focus(on, event) {
        box.classList.toggle("focus", on);
        item.classList.toggle("hot", on);
        if (path) { path.classList.toggle("hot", on); }
        if (on && event && path) { showTip(event, slice, total, format); } else { hideTip(); }
      }
      item.addEventListener("mouseenter", function () { focus(true); });
      item.addEventListener("mouseleave", function () { focus(false); });
      if (path) {
        path.addEventListener("mouseenter", function (event) { focus(true, event); });
        path.addEventListener("mousemove", function (event) { showTip(event, slice, total, format); });
        path.addEventListener("mouseleave", function () { focus(false); });
      }
    });
    var center = svgEl("text", { x: 90, y: 88, "text-anchor": "middle", "class": "center-value" });
    center.textContent = format(total);
    svg.appendChild(center);
    var label = svgEl("text", { x: 90, y: 106, "text-anchor": "middle", "class": "center-label" });
    for (var m = 0; m < PIE_METRICS.length; m++) {
      if (PIE_METRICS[m][0] === metric) { label.textContent = t(PIE_METRICS[m][1]); }
    }
    svg.appendChild(label);
    svg.setAttribute("aria-label", label.textContent + ": " + slices.map(function (slice) {
      return slice.name + " " + format(slice.value);
    }).join(", "));
    box.appendChild(svg);

    var side = el("div", "viz-side");
    var switcher = el("div", "viz-metrics");
    PIE_METRICS.forEach(function (option) {
      var button = el("button", option[0] === metric ? "on" : "", t(option[1]));
      button.type = "button";
      button.addEventListener("click", function () { hideTip(); renderPie(box, entries, option[0]); });
      switcher.appendChild(button);
    });
    side.appendChild(switcher);
    side.appendChild(legend);
    box.appendChild(side);
  }

  // Rate limit inputs show the key's base rate limits; blank means inherit.
  function fillRate(rate) {
    rate = rate || {};
    document.getElementById("r_rpm").value = rate.rpm ? rate.rpm : "";
    document.getElementById("r_tpm").value = rate.tpm ? rate.tpm : "";
    document.getElementById("r_conc").value = rate.concurrency ? rate.concurrency : "";
    document.getElementById("r_queue").value = rate.queue_seconds ? rate.queue_seconds : "";
  }

  function readRate() {
    return {
      rpm: Math.trunc(numberOf("r_rpm")),
      tpm: Math.trunc(numberOf("r_tpm")),
      concurrency: Math.trunc(numberOf("r_conc")),
      queue_seconds: Math.trunc(numberOf("r_queue"))
    };
  }

  var DAY_KEYS = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
  var schedWindows = [];
  var schedInherited = null;

  // expandDays mirrors parseDays on the server: names, ranges and the
  // weekdays / weekends / all shortcuts, with an empty list meaning every day.
  function expandDays(days) {
    var picked = [false, false, false, false, false, false, false];
    function index(name) { return DAY_KEYS.indexOf(String(name).slice(0, 3)); }
    (days || []).forEach(function (raw) {
      var day = String(raw).toLowerCase().trim();
      if (!day) { return; }
      if (day === "*" || day === "all" || day === "daily" || day === "everyday") { picked = picked.map(function () { return true; }); return; }
      if (day === "weekdays") { [0, 1, 2, 3, 4].forEach(function (i) { picked[i] = true; }); return; }
      if (day === "weekends") { picked[5] = true; picked[6] = true; return; }
      var parts = day.split("-");
      if (parts.length === 2) {
        var first = index(parts[0]);
        var last = index(parts[1]);
        if (first < 0 || last < 0) { return; }
        for (var i = first; ; i = (i + 1) % 7) { picked[i] = true; if (i === last) { break; } }
        return;
      }
      if (index(day) >= 0) { picked[index(day)] = true; }
    });
    if (picked.indexOf(true) === -1) { picked = picked.map(function () { return true; }); }
    return picked;
  }

  function windowState(window) {
    var limit = window.limits || {};
    var dim = "";
    var value = "";
    if (limit.cost_usd > 0) { dim = "cost_usd"; value = limit.cost_usd; }
    else if (limit.tokens > 0) { dim = "tokens"; value = limit.tokens; }
    else if (limit.requests > 0) { dim = "requests"; value = limit.requests; }
    var rate = window.rate_limits || {};
    function clock(value) { return value === "24:00" ? "00:00" : (value || ""); }
    return {
      name: window.name || "",
      days: expandDays(window.days),
      start: clock(window.start),
      end: clock(window.end),
      block: !!window.block,
      dim: dim,
      value: value,
      rpm: rate.rpm || "",
      tpm: rate.tpm || "",
      conc: rate.concurrency || ""
    };
  }

  function scheduleSummary(sched) {
    if (!sched) { return ""; }
    return (sched.windows || []).length + " " + t("schedWindows") + ", " + t("schedOutside") + ": " +
      (sched.outside === "block" ? t("schedBlockAll") : t("schedAllow"));
  }

  function syncSchedule() {
    var own = document.getElementById("schedOwn").checked;
    document.getElementById("schedBody").hidden = !own;
    var source = document.getElementById("schedSource");
    if (own) { source.textContent = t("schedFromRuntime"); return; }
    if (!schedInherited || !editing.schedule_source || editing.schedule_source === "runtime") {
      source.textContent = t("schedNone");
      return;
    }
    source.textContent = (editing.schedule_source === "config" ? t("schedFromConfig") : t("schedFromDefault")) +
      " (" + scheduleSummary(schedInherited) + ")";
  }

  function fillSchedule(item) {
    var own = item.schedule_source === "runtime";
    // The inherited schedule seeds the editor when the operator opts in.
    schedInherited = own ? null : item.schedule;
    var start = item.schedule || { outside: "allow", windows: [] };
    schedWindows = (start.windows || []).map(windowState);
    document.getElementById("schedOutside").value = start.outside === "block" ? "block" : "allow";
    document.getElementById("schedOwn").checked = own;
    renderWindows();
    syncSchedule();
  }

  function numberInput(value, placeholder, onInput) {
    var input = el("input");
    input.type = "number";
    input.step = "any";
    input.placeholder = placeholder;
    input.value = value === 0 ? "" : value;
    input.addEventListener("input", function () { onInput(input.value); });
    return input;
  }

  function renderWindows() {
    var list = document.getElementById("winList");
    list.innerHTML = "";
    var dayNames = t("dayNames").split(",");
    schedWindows.forEach(function (win, index) {
      var card = el("div", "win");
      var row = el("div", "win-row");
      var name = el("input");
      name.type = "text";
      name.placeholder = t("winName");
      name.value = win.name;
      name.addEventListener("input", function () { win.name = name.value; });
      row.appendChild(name);
      ["start", "end"].forEach(function (field) {
        var clock = el("input");
        clock.type = "time";
        clock.value = win[field];
        clock.addEventListener("input", function () { win[field] = clock.value; });
        row.appendChild(clock);
      });
      var action = el("select");
      [["limit", "winLimit"], ["block", "winBlock"]].forEach(function (option) {
        var node = el("option", null, t(option[1]));
        node.value = option[0];
        action.appendChild(node);
      });
      action.value = win.block ? "block" : "limit";
      action.addEventListener("change", function () {
        win.block = action.value === "block";
        limitsBox.hidden = win.block;
      });
      row.appendChild(action);
      // Order decides which window wins where two overlap.
      var buttons = el("span", "rowbtns");
      if (index > 0) {
        buttons.appendChild(actionButton(t("moveUp"), function () {
          schedWindows.splice(index - 1, 0, schedWindows.splice(index, 1)[0]);
          renderWindows();
        }, "ghost"));
      }
      buttons.appendChild(actionButton(t("remove"), function () {
        schedWindows.splice(index, 1);
        renderWindows();
      }, "danger"));
      row.appendChild(buttons);
      card.appendChild(row);

      var days = el("div", "win-days");
      DAY_KEYS.forEach(function (key, day) {
        var chip = el("button", win.days[day] ? "on" : "", dayNames[day]);
        chip.type = "button";
        chip.setAttribute("aria-pressed", win.days[day] ? "true" : "false");
        chip.addEventListener("click", function () {
          win.days[day] = !win.days[day];
          chip.className = win.days[day] ? "on" : "";
          chip.setAttribute("aria-pressed", win.days[day] ? "true" : "false");
        });
        days.appendChild(chip);
      });
      card.appendChild(days);

      var limitsBox = el("div", "win-limits");
      limitsBox.hidden = win.block;
      limitsBox.appendChild(el("span", "period", t("winQuota")));
      var dim = el("select");
      [["", "winNoQuota"], ["cost_usd", "limCost"], ["tokens", "limTokens"], ["requests", "limRequests"]].forEach(function (option) {
        var node = el("option", null, t(option[1]));
        node.value = option[0];
        dim.appendChild(node);
      });
      dim.value = win.dim;
      var value = numberInput(win.value, "", function (next) { win.value = next; });
      value.disabled = !win.dim;
      dim.addEventListener("change", function () {
        win.dim = dim.value;
        value.disabled = !win.dim;
        if (!win.dim) { value.value = ""; win.value = ""; }
      });
      limitsBox.appendChild(dim);
      limitsBox.appendChild(value);
      limitsBox.appendChild(el("span", "period", t("winRate")));
      var rates = el("div", "rates");
      rates.appendChild(numberInput(win.rpm, "RPM", function (next) { win.rpm = next; }));
      rates.appendChild(numberInput(win.tpm, "TPM", function (next) { win.tpm = next; }));
      rates.appendChild(numberInput(win.conc, t("tagConc"), function (next) { win.conc = next; }));
      limitsBox.appendChild(rates);
      card.appendChild(limitsBox);
      list.appendChild(card);
    });
  }

  function toNumber(raw) {
    var parsed = Number(String(raw).trim());
    return String(raw).trim() && !isNaN(parsed) ? parsed : 0;
  }

  // readSchedule builds the schedule payload, or throws a message to show.
  function readSchedule() {
    return {
      outside: document.getElementById("schedOutside").value,
      windows: schedWindows.map(function (win) {
        if (!win.start || !win.end) { throw new Error(t("winNeedTime")); }
        var days = DAY_KEYS.filter(function (key, day) { return win.days[day]; });
        if (!days.length) { throw new Error(t("winNeedDays")); }
        var limit = { tokens: 0, requests: 0, cost_usd: 0 };
        if (win.dim && !win.block) { limit[win.dim] = toNumber(win.value); }
        return {
          name: win.name.trim(),
          days: days.length === 7 ? [] : days,
          start: win.start,
          end: win.end,
          block: win.block,
          limits: limit,
          rate_limits: win.block ? { rpm: 0, tpm: 0, concurrency: 0 } : {
            rpm: Math.trunc(toNumber(win.rpm)),
            tpm: Math.trunc(toNumber(win.tpm)),
            concurrency: Math.trunc(toNumber(win.conc))
          }
        };
      })
    };
  }

  function openEditor(item) {
    editing = item;
    document.getElementById("editTitle").textContent =
      t("editKey") + ": " + (item.label || item.hint || item.id);
    document.getElementById("editSub").textContent = item.id;
    document.getElementById("editError").textContent = "";
    fillLimit("d", item.periods.daily.limits);
    fillLimit("m", item.periods.monthly.limits);
    fillLimit("t", item.periods.total.limits);
    fillRate(item.base_rate_limits);
    fillSchedule(item);
    document.getElementById("keyNote").value = item.note || "";
    keyBlockedPatterns = (item.own_blocked_models || []).slice();
    document.getElementById("keyBlockedInput").value = "";
    renderKeyBlocked();
    keyMappings = (item.own_model_mappings || []).map(function (rule) {
      return { from: rule.from, to: rule.to, provider: rule.provider || "" };
    });
    ["keyMapTo", "keyMapProvider"].forEach(function (id) {
      document.getElementById(id).value = "";
    });
    renderKeyMappings();
    fillFallbacks(item);
    fillSuggestions(item);
    resetMapFrom();
    keyAccounts = (item.accounts || []).slice();
    document.getElementById("keyAccountInput").value = "";
    document.getElementById("strictAccounts").checked = !!item.strict_accounts;
    renderKeyAccounts();
    // The account list feeds the suggestions and loads lazily like /pricing.
    if (!lastAccounts) { loadAccounts(); }
    // Recognized models and price patterns come from /pricing, which loads
    // lazily; fetch it so the dropdown lists them on first open too.
    if (!lastPricing) {
      loadPricing().then(function () { if (editing === item) { fillSuggestions(item); } });
    }
    document.getElementById("disabledFlag").checked = !!item.disabled;
    editor.showModal();
  }

  // SHA-256 mirrors keyID on the server. crypto.subtle only exists in secure
  // contexts, and the panel is often opened over plain http, so a compact pure
  // implementation backs it up.
  var SHA_K = [
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2
  ];

  function sha256Pure(bytes) {
    var h = [0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19];
    var length = bytes.length;
    var padded = new Uint8Array(((length + 9 + 63) >> 6) << 6);
    padded.set(bytes);
    padded[length] = 0x80;
    var bits = length * 8;
    var end = padded.length;
    padded[end - 4] = (bits >>> 24) & 0xff;
    padded[end - 3] = (bits >>> 16) & 0xff;
    padded[end - 2] = (bits >>> 8) & 0xff;
    padded[end - 1] = bits & 0xff;
    padded[end - 5] = Math.floor(length / 0x20000000) & 0xff;
    var w = new Array(64);
    function rotr(x, n) { return (x >>> n) | (x << (32 - n)); }
    for (var offset = 0; offset < end; offset += 64) {
      for (var i = 0; i < 16; i++) {
        var j = offset + i * 4;
        w[i] = (padded[j] << 24) | (padded[j + 1] << 16) | (padded[j + 2] << 8) | padded[j + 3];
      }
      for (i = 16; i < 64; i++) {
        var s0 = rotr(w[i - 15], 7) ^ rotr(w[i - 15], 18) ^ (w[i - 15] >>> 3);
        var s1 = rotr(w[i - 2], 17) ^ rotr(w[i - 2], 19) ^ (w[i - 2] >>> 10);
        w[i] = (w[i - 16] + s0 + w[i - 7] + s1) | 0;
      }
      var a = h[0], b = h[1], c = h[2], d = h[3], e = h[4], f = h[5], g = h[6], k = h[7];
      for (i = 0; i < 64; i++) {
        var t1 = (k + (rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25)) + ((e & f) ^ (~e & g)) + SHA_K[i] + w[i]) | 0;
        var t2 = ((rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22)) + ((a & b) ^ (a & c) ^ (b & c))) | 0;
        k = g; g = f; f = e; e = (d + t1) | 0; d = c; c = b; b = a; a = (t1 + t2) | 0;
      }
      h[0] = (h[0] + a) | 0; h[1] = (h[1] + b) | 0; h[2] = (h[2] + c) | 0; h[3] = (h[3] + d) | 0;
      h[4] = (h[4] + e) | 0; h[5] = (h[5] + f) | 0; h[6] = (h[6] + g) | 0; h[7] = (h[7] + k) | 0;
    }
    var out = new Uint8Array(32);
    for (i = 0; i < 8; i++) {
      out[i * 4] = (h[i] >>> 24) & 0xff;
      out[i * 4 + 1] = (h[i] >>> 16) & 0xff;
      out[i * 4 + 2] = (h[i] >>> 8) & 0xff;
      out[i * 4 + 3] = h[i] & 0xff;
    }
    return out;
  }

  function hex(bytes) {
    var out = "";
    for (var i = 0; i < bytes.length; i++) { out += (bytes[i] < 16 ? "0" : "") + bytes[i].toString(16); }
    return out;
  }

  // keyIDOf computes the same truncated digest the plugin uses as a key id.
  function keyIDOf(key) {
    var bytes = new TextEncoder().encode(String(key).trim());
    var subtle = window.crypto && window.crypto.subtle;
    if (subtle) {
      return subtle.digest("SHA-256", bytes).then(function (digest) {
        return hex(new Uint8Array(digest)).slice(0, 16);
      });
    }
    return Promise.resolve(hex(sha256Pure(bytes)).slice(0, 16));
  }

  // generateKey returns a fresh client key: "sk-" and 48 random hex digits.
  function generateKey() {
    var bytes = new Uint8Array(24);
    window.crypto.getRandomValues(bytes);
    return "sk-" + hex(bytes);
  }

  var rotateDialog = document.getElementById("rotateDialog");
  // rotating holds the keys of the open rotate dialog and, once it ran, one
  // result per key: state is "ok", "migrate" (the host took the new key but
  // the plugin state did not move yet), "notfound" or "host".
  var rotating = null;

  function rotateStage(stage) {
    var done = stage === "done" || stage === "retry";
    document.getElementById("rotateConfirm").hidden = done;
    document.getElementById("rotateTargets").hidden = done || rotating.items.length < 2;
    document.getElementById("rotateResult").hidden = !done;
    document.getElementById("rotateGo").hidden = done;
    document.getElementById("rotateCancel").hidden = done;
    document.getElementById("rotateRetry").hidden = stage !== "retry";
    document.getElementById("rotateDone").hidden = !done;
  }

  function openRotate(items) {
    if (!items.length) { return; }
    rotating = { items: items, results: [] };
    // A batch lists "name: key" lines, which need the wider dialog.
    rotateDialog.classList.toggle("wide", items.length > 1);
    document.getElementById("rotateTitle").textContent = dialogTitle(t("rotateTitle"), items);
    fillTargets("rotateTargets", items, false);
    document.getElementById("rotateError").textContent = "";
    document.getElementById("rotateConfigured").hidden = true;
    document.getElementById("newKeyOut").value = "";
    document.getElementById("copyNewKey").textContent = t("copy");
    var go = document.getElementById("rotateGo");
    go.disabled = false;
    go.textContent = t("rotateGo");
    rotateStage("confirm");
    rotateDialog.showModal();
  }

  function hostRequest(method, path, payload) {
    var init = { method: method, headers: headers(), cache: "no-store" };
    if (payload !== undefined) { init.body = JSON.stringify(payload); }
    return fetch(hostBase + path, init).then(function (response) {
      if (!response.ok) { return requestFailed(response); }
      return response.json();
    });
  }

  // hostKeyMap maps plugin ids to the raw keys of the host's api-keys list.
  function hostKeyMap() {
    return hostRequest("GET", "/api-keys").then(function (data) {
      var keys = (data && data["api-keys"]) || [];
      return Promise.all(keys.map(keyIDOf)).then(function (ids) {
        var out = {};
        ids.forEach(function (id, index) { out[id] = keys[index]; });
        return out;
      });
    });
  }

  // sequence runs step over the items one after another, so host edits never
  // race each other.
  function sequence(items, step) {
    return items.reduce(function (chain, item) {
      return chain.then(function () { return step(item); });
    }, Promise.resolve());
  }

  // migrateResult moves the plugin state of one rotated key. It runs after the
  // host already accepted the new key, so a failure keeps the key on screen
  // and offers a retry instead of losing it.
  function migrateResult(result) {
    return post("/rotate", { id: result.item.id, new_key: result.newKey })
      .then(function (data) {
        result.state = "ok";
        result.configured = !!data.configured;
        result.error = "";
      })
      .catch(function (err) {
        result.state = "migrate";
        result.error = err.message;
      });
  }

  function showRotateResults() {
    var results = rotating.results;
    var single = rotating.items.length === 1;
    var issued = results.filter(function (result) { return result.newKey; });
    var output = document.getElementById("newKeyOut");
    output.value = issued.map(function (result) {
      return single ? result.newKey : displayName(result.item) + ": " + result.newKey;
    }).join("\n");
    // The spare row keeps a horizontal scrollbar from hiding the last line.
    output.rows = single ? 1 : Math.min(issued.length, 8) + 1;

    var configured = results.filter(function (result) { return result.state === "ok" && result.configured; });
    var configuredEl = document.getElementById("rotateConfigured");
    configuredEl.hidden = !configured.length;
    configuredEl.textContent = t("rotateConfigured") + (single ? "" : " (" +
      configured.map(function (result) { return displayName(result.item); }).join(", ") + ")");

    var errors = results.filter(function (result) { return result.state !== "ok"; }).map(function (result) {
      var prefix = single ? "" : displayName(result.item) + ": ";
      if (result.state === "notfound") { return prefix + t("rotateNotFound"); }
      if (result.state === "host") { return prefix + t("rotateHostFailed") + ": " + result.error; }
      return prefix + t("rotateMigrateFailed") + ": " + result.error;
    });
    document.getElementById("rotateError").textContent = errors.join("\n");
    // Nothing changed in the host when no key was issued, so the dialog stays
    // on the confirm step and the operator can try again.
    if (issued.length) {
      rotateStage(results.some(function (result) { return result.state === "migrate"; }) ? "retry" : "done");
    }
    if (results.some(function (result) { return result.state === "ok"; })) {
      setStatus(t("rotated"), false, "ok");
      load();
    }
  }

  function retryMigration() {
    var pending = rotating.results.filter(function (result) { return result.state === "migrate"; });
    document.getElementById("rotateError").textContent = "";
    sequence(pending, migrateResult).then(showRotateResults);
  }

  function runRotation() {
    if (!rotating) { return; }
    var go = document.getElementById("rotateGo");
    var errorEl = document.getElementById("rotateError");
    go.disabled = true;
    go.textContent = t("rotating");
    errorEl.textContent = "";
    rotating.results = [];
    function idle() {
      go.disabled = false;
      go.textContent = t("rotateGo");
    }
    hostKeyMap()
      .then(function (known) {
        return sequence(rotating.items, function (item) {
          var result = { item: item, newKey: "", state: "", error: "" };
          rotating.results.push(result);
          var oldKey = known[item.id];
          if (!oldKey) { result.state = "notfound"; return; }
          var newKey = generateKey();
          return hostRequest("PATCH", "/api-keys", { old: oldKey, "new": newKey })
            .then(function () {
              result.newKey = newKey;
              return migrateResult(result);
            }, function (err) {
              result.state = "host";
              result.error = err.message;
            });
        });
      })
      .then(function () { idle(); showRotateResults(); })
      .catch(function (err) {
        idle();
        errorEl.textContent = t("rotateHostFailed") + ": " + err.message;
      });
  }

  function openDelete(items) {
    if (!items.length) { return; }
    deleting = items;
    document.getElementById("deleteTitle").textContent = dialogTitle(t("deleteTitle"), items);
    fillTargets("deleteTargets", items, true);
    var fromHost = document.getElementById("deleteFromHost");
    fromHost.checked = true;
    fromHost.disabled = false;
    document.getElementById("deleteError").textContent = "";
    document.getElementById("deleteReport").hidden = true;
    var confirm = document.getElementById("confirmDelete");
    confirm.hidden = false;
    confirm.disabled = false;
    confirm.textContent = t("deleteBtn");
    document.getElementById("deleteCancel").hidden = false;
    document.getElementById("deleteDone").hidden = true;
    deleteDialog.showModal();
  }

  // runDelete removes each key from the host first and then drops the plugin
  // records. A key the host refused to remove keeps its record, since it still
  // works and keeps being metered.
  function runDelete() {
    var items = deleting.slice();
    if (!items.length) { return; }
    var fromHost = document.getElementById("deleteFromHost").checked;
    var confirm = document.getElementById("confirmDelete");
    var errorEl = document.getElementById("deleteError");
    confirm.disabled = true;
    confirm.textContent = t("deleting");
    errorEl.textContent = "";
    var notes = [];
    var drop = [];
    var names = {};
    items.forEach(function (item) { names[item.id] = displayName(item); });

    (fromHost ? hostKeyMap() : Promise.resolve({}))
      .then(function (known) {
        return sequence(items, function (item) {
          if (!fromHost) { drop.push(item.id); return; }
          var raw = known[item.id];
          if (!raw) {
            notes.push(names[item.id] + ": " + t("deleteNotInHost"));
            drop.push(item.id);
            return;
          }
          return hostRequest("DELETE", "/api-keys?value=" + encodeURIComponent(raw))
            .then(function () { drop.push(item.id); }, function (err) {
              notes.push(names[item.id] + ": " + t("deleteHostFailed") + " (" + err.message + ")");
            });
        });
      })
      .then(function () {
        if (!drop.length) { return null; }
        return post("/delete", { ids: drop });
      })
      .then(function (result) {
        drop.forEach(function (id) { delete selected[id]; });
        ((result && result.configured) || []).forEach(function (id) {
          notes.push(names[id] + ": " + t("deleteConfigured"));
        });
        if (drop.length) {
          setStatus(t("deleted"), false, "ok");
          load();
        }
        if (!notes.length) {
          deleteDialog.close();
          return;
        }
        var report = document.getElementById("deleteReport");
        report.innerHTML = "";
        notes.forEach(function (text) { report.appendChild(el("li", null, text)); });
        report.hidden = false;
        document.getElementById("deleteFromHost").disabled = true;
        confirm.hidden = true;
        document.getElementById("deleteCancel").hidden = true;
        document.getElementById("deleteDone").hidden = false;
      })
      .catch(function (err) {
        confirm.disabled = false;
        confirm.textContent = t("deleteBtn");
        errorEl.textContent = t("deleteFailed") + ": " + err.message;
      });
  }

  function copyNewKey() {
    var input = document.getElementById("newKeyOut");
    var button = document.getElementById("copyNewKey");
    function copied() { button.textContent = t("copied"); }
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(input.value).then(copied, function () {
        input.select();
        if (document.execCommand("copy")) { copied(); }
      });
      return;
    }
    input.focus();
    input.select();
    if (document.execCommand("copy")) { copied(); }
  }

  var TABS = {
    keys: ["tabKeys", "panelKeys"],
    rules: ["tabRules", "panelRules"],
    accounts: ["tabAccounts", "panelAccounts"],
    pricing: ["tabPricing", "panelPricing"]
  };

  function switchTab(name) {
    activeTab = name;
    Object.keys(TABS).forEach(function (key) {
      var tab = document.getElementById(TABS[key][0]);
      tab.className = key === name ? "tab on" : "tab";
      tab.setAttribute("aria-selected", key === name ? "true" : "false");
      document.getElementById(TABS[key][1]).hidden = key !== name;
    });
    if (name === "pricing" && !lastPricing) { loadPricing(); }
    if (name === "accounts") { loadAccounts(); }
  }

  function stopRefresh() {
    if (timer) { clearInterval(timer); timer = null; }
  }

  function scheduleRefresh() {
    stopRefresh();
    var seconds = Number(document.getElementById("interval").value);
    if (seconds > 0 && normalizeKey()) { timer = setInterval(load, seconds * 1000); }
  }

  document.getElementById("langEn").addEventListener("click", function () { setLanguage("en"); });
  document.getElementById("langZh").addEventListener("click", function () { setLanguage("zh"); });
  Object.keys(TABS).forEach(function (key) {
    document.getElementById(TABS[key][0]).addEventListener("click", function () { switchTab(key); });
  });

  // Every dialog closes through buttons marked data-close. They are type=button,
  // so pressing Enter in a field never submits and closes a dialog by accident.
  var closers = document.querySelectorAll("[data-close]");
  for (var c = 0; c < closers.length; c++) {
    closers[c].addEventListener("click", function () { this.closest("dialog").close(); });
  }

  document.getElementById("openSettings").addEventListener("click", openSettings);
  document.getElementById("rotateGo").addEventListener("click", runRotation);
  document.getElementById("rotateRetry").addEventListener("click", retryMigration);
  document.getElementById("confirmDelete").addEventListener("click", runDelete);
  document.getElementById("schedOwn").addEventListener("change", function () {
    // Opting in starts from whatever schedule applied so far.
    if (this.checked && schedInherited && !schedWindows.length) {
      schedWindows = (schedInherited.windows || []).map(windowState);
      document.getElementById("schedOutside").value = schedInherited.outside === "block" ? "block" : "allow";
      renderWindows();
    }
    syncSchedule();
  });
  document.getElementById("addWindow").addEventListener("click", function () {
    schedWindows.push(windowState({ start: "09:00", end: "18:00", days: ["weekdays"] }));
    renderWindows();
  });
  var observedPie = document.getElementById("observedPie");
  observedPie.checked = observedPieOn();
  observedPie.addEventListener("change", function () {
    try { localStorage.setItem(PIE_STORAGE, observedPie.checked ? "1" : "0"); } catch (err) { /* ignore */ }
    if (lastPricing) { renderObservedPie(lastPricing); }
  });
  ["modelsDialog", "editor"].forEach(function (id) {
    document.getElementById(id).addEventListener("close", hideTip);
  });
  document.getElementById("selectAll").addEventListener("change", function () {
    var pick = this.checked;
    shownKeys.forEach(function (item) {
      if (pick) { selected[item.id] = true; } else { delete selected[item.id]; }
    });
    updateSelection();
    var boxes = rowsEl.querySelectorAll("input.pick");
    for (var i = 0; i < boxes.length; i++) { boxes[i].checked = pick; }
  });
  document.getElementById("batchPause").addEventListener("click", function () { batchPause(true); });
  document.getElementById("batchEnable").addEventListener("click", function () { batchPause(false); });
  document.getElementById("batchReset").addEventListener("click", function () { resetKey(selectedItems()); });
  document.getElementById("batchRotate").addEventListener("click", function () { openRotate(selectedItems()); });
  document.getElementById("batchDelete").addEventListener("click", function () { openDelete(selectedItems()); });
  document.getElementById("batchClear").addEventListener("click", clearSelection);
  document.getElementById("copyNewKey").addEventListener("click", copyNewKey);
  document.getElementById("newKeyOut").addEventListener("focus", function () { this.select(); });
  document.getElementById("connect").addEventListener("click", function () {
    settingsDialog.close();
    load();
  });
  keyInput.addEventListener("keydown", function (event) {
    if (event.key === "Enter") {
      event.preventDefault();
      settingsDialog.close();
      load();
    }
  });

  filterInput.addEventListener("input", function () {
    if (lastUsage) { renderRows(lastUsage); }
  });

  document.getElementById("revealKey").addEventListener("click", function () {
    var shown = keyInput.type === "text";
    keyInput.type = shown ? "password" : "text";
    this.textContent = shown ? t("show") : t("hide");
  });

  document.getElementById("reload").addEventListener("click", load);
  document.getElementById("interval").addEventListener("change", scheduleRefresh);

  document.getElementById("addPrice").addEventListener("click", function () {
    if (priceRowsEl.children.length === 1 && !priceRowsEl.querySelectorAll("input").length) {
      priceRowsEl.innerHTML = "";
    }
    addPriceRow(null);
  });

  document.getElementById("savePricing").addEventListener("click", function () {
    post("/pricing", collectPricing())
      .then(function (data) {
        lastPricing = data;
        renderPricing(data);
        renderObserved(data);
        setStatus(t("saved"));
      })
      .catch(function (err) { handleFailure(err, t("saveFailed")); });
  });

  document.getElementById("clearPricing").addEventListener("click", function () {
    post("/pricing", { clear: true })
      .then(function (data) {
        lastPricing = data;
        renderPricing(data);
        renderObserved(data);
        setStatus(t("saved"));
      })
      .catch(function (err) { handleFailure(err, t("saveFailed")); });
  });

  document.getElementById("saveBlocked").addEventListener("click", function () {
    post("/models", { blocked_models: lines(blockedEl.value) })
      .then(function () { setStatus(t("saved")); load(); loadPricing(); })
      .catch(function (err) { handleFailure(err, t("saveFailed")); });
  });

  document.getElementById("clearBlocked").addEventListener("click", function () {
    post("/models", { clear: true })
      .then(function () { setStatus(t("saved")); load(); loadPricing(); })
      .catch(function (err) { handleFailure(err, t("saveFailed")); });
  });

  ["d", "m", "t"].forEach(function (prefix) {
    document.getElementById(prefix + "_dim").addEventListener("change", function () {
      syncLimitInput(prefix);
    });
  });

  document.getElementById("keyBlockedAdd").addEventListener("click", addKeyBlocked);
  document.getElementById("keyBlockedInput").addEventListener("keydown", function (event) {
    if (event.key === "Enter") { event.preventDefault(); addKeyBlocked(); }
  });

  document.getElementById("keyMapAdd").addEventListener("click", addKeyMapping);
  document.getElementById("keyMapFromSelect").addEventListener("change", syncMapFrom);
  ["keyMapFrom", "keyMapTo", "keyMapProvider"].forEach(function (id) {
    document.getElementById(id).addEventListener("keydown", function (event) {
      if (event.key === "Enter") { event.preventDefault(); addKeyMapping(); }
    });
  });

  document.getElementById("addMapping").addEventListener("click", function () {
    setMappingsDirty(true);
    addMappingRow(null);
  });
  mappingRowsEl.addEventListener("input", function () { setMappingsDirty(true); });
  mappingRowsEl.addEventListener("click", function (event) {
    if (event.target.tagName === "BUTTON") { setMappingsDirty(true); }
  });

  document.getElementById("saveMappings").addEventListener("click", function () {
    post("/mappings", { model_mappings: collectMappings() })
      .then(function () { setMappingsDirty(false); setStatus(t("saved")); load(); })
      .catch(function (err) { handleFailure(err, t("saveFailed")); });
  });

  document.getElementById("fallbackOwn").addEventListener("change", syncFallbacks);

  document.getElementById("keyAccountAdd").addEventListener("click", addKeyAccount);
  document.getElementById("keyAccountInput").addEventListener("keydown", function (event) {
    if (event.key === "Enter") { event.preventDefault(); addKeyAccount(); }
  });

  document.getElementById("addAccountRule").addEventListener("click", function () {
    addAccountRuleRow(null);
    setAccountRulesDirty(true);
  });

  document.getElementById("saveAccountRules").addEventListener("click", function () {
    post("/accounts", { rules: collectAccountRules() })
      .then(function (data) { setAccountRulesDirty(false); renderAccountsData(data); setStatus(t("saved")); })
      .catch(function (err) { handleFailure(err, t("saveFailed")); });
  });

  document.getElementById("clearAccountRules").addEventListener("click", function () {
    post("/accounts", { clear: true })
      .then(function (data) { setAccountRulesDirty(false); renderAccountsData(data); setStatus(t("saved")); })
      .catch(function (err) { handleFailure(err, t("saveFailed")); });
  });

  document.getElementById("saveFallbacks").addEventListener("click", function () {
    post("/fallbacks", { fallback_models: lines(fallbacksEl.value) })
      .then(function () { setStatus(t("saved")); load(); })
      .catch(function (err) { handleFailure(err, t("saveFailed")); });
  });

  document.getElementById("clearFallbacks").addEventListener("click", function () {
    post("/fallbacks", { clear: true })
      .then(function () { setStatus(t("saved")); load(); })
      .catch(function (err) { handleFailure(err, t("saveFailed")); });
  });

  document.getElementById("clearMappings").addEventListener("click", function () {
    post("/mappings", { clear: true })
      .then(function () { setMappingsDirty(false); setStatus(t("saved")); load(); })
      .catch(function (err) { handleFailure(err, t("saveFailed")); });
  });

  document.getElementById("confirmReset").addEventListener("click", function () {
    if (!resetting.length) { return; }
    post("/reset", {
      ids: resetting.map(function (item) { return item.id; }),
      period: document.getElementById("resetSelect").value
    })
      .then(function () { resetDialog.close(); load(); })
      .catch(function (err) {
        document.getElementById("resetError").textContent = t("resetFailed") + ": " + err.message;
      });
  });

  document.getElementById("saveEdit").addEventListener("click", function () {
    if (!editing) { return; }
    var payload = {
      id: editing.id,
      disabled: document.getElementById("disabledFlag").checked,
      blocked_models: keyBlockedPatterns.slice(),
      model_mappings: keyMappings.slice(),
      note: document.getElementById("keyNote").value.trim(),
      limits: {
        daily: readLimit("d"),
        monthly: readLimit("m"),
        total: readLimit("t")
      },
      rate_limits: readRate(),
      accounts: keyAccounts.slice(),
      strict_accounts: document.getElementById("strictAccounts").checked
    };
    if (document.getElementById("schedOwn").checked) {
      try { payload.schedule = readSchedule(); } catch (err) {
        document.getElementById("editError").textContent = err.message;
        return;
      }
    } else {
      payload.clear_schedule = true;
    }
    if (document.getElementById("fallbackOwn").checked) {
      payload.fallback_models = lines(document.getElementById("keyFallbacks").value);
    } else {
      payload.clear_fallback_models = true;
    }
    post("/limits", payload).then(function () {
      editor.close();
      load();
    }).catch(function (err) {
      document.getElementById("editError").textContent = t("saveFailed") + ": " + err.message;
    });
  });

  document.getElementById("clearOverrides").addEventListener("click", function () {
    if (!editing) { return; }
    post("/limits", { id: editing.id, clear: true })
      .then(function () { editor.close(); load(); })
      .catch(function (err) {
        document.getElementById("editError").textContent = t("saveFailed") + ": " + err.message;
      });
  });

  try {
    var saved = sessionStorage.getItem(KEY_STORAGE);
    if (saved) { keyInput.value = saved; }
  } catch (err) { /* storage may be unavailable */ }

  applyStatic();
  syncTheme();
  if (normalizeKey()) { load(); } else { showMessage(t("enterKey")); openSettings(); }
})();
`
