# 模型 — 厂商关系表

> 维护：MOI TaaS。本文档列出主流大模型系列、对应厂商、核心标签与开源状态，作为渠道接入和模型管理的参考。
> Logo 资源放在本目录下（`<brand>.svg`），命名约定见末尾「Logo 资源总览」。

---

## 零、模型类型分类（8 类）

> 设计原则：**每一类内部 API endpoint + 计费单位完全一致**。

| # | 类型 | type 值 | API endpoint | 计费单位 | 包含子能力 | 示例模型 |
|---|---|---|---|---|---|---|
| 1 | **对话** | `chat` | `POST /v1/chat/completions` | cr/M tokens（4 段：普通输入 / 缓存创建 / 缓存命中 / 输出） | 纯文本对话 + 视觉多模态(VLM) + VLM 系 OCR | Qwen-Max / Claude / GPT-4o / Gemini / Qwen-VL / qwen-vl-ocr / deepseek-ocr |
| 2 | **OCR** | `ocr` | `POST /v1/ocr` | cr/页 | 传统 OCR（非 VLM 系） | PaddleOCR |
| 3 | **文生图** | `image` | `POST /v1/images/generations` | cr/张 | 文生图 / 图像编辑 | Wan2.7-Image / SD / Midjourney / Kolors / Z-Image / qwen-image / FLUX |
| 4 | **文生视频** | `video` | `POST /v1/videos/generations` | cr/秒 | 文/图驱动生视频 | Wan2.2-i2v / Wan2.2-t2v / Kling |
| 5 | **文转音 TTS** | `tts` | `POST /v1/audio/speech` | cr/万字符 | 语音合成 | speech-02-hd / cosyvoice / sambert / f5-tts / kokoro |
| 6 | **音转文 ASR** | `asr` | `POST /v1/audio/transcriptions` | cr/分钟 | 语音识别 | telespeech / qwen3-asr-flash |
| 7 | **嵌入** | `embed` | `POST /v1/embeddings` | cr/M tokens（输入） | 文本嵌入 | bge / bce / qwen3-embedding / text-embedding-3 |
| 8 | **重排序** | `rerank` | `POST /v1/rerank` | cr/M tokens（输入） | 语义重排 | bge-reranker / bce-reranker / qwen3-reranker |

---

## 一、对话（chat）

> 含纯文本对话、视觉多模态（VLM）、VLM 系 OCR。统一走 `/v1/chat/completions`，按 token 计费。

| Logo | 模型系列 | 代表版本 | 厂商 | 核心标签与亮点 | 开源状态 |
|---|---|---|---|---|---|
| <img src="./deepseek.svg" width="32"> | **DeepSeek** | DeepSeek V3, DeepSeek R1, deepseek-ocr（VLM-OCR） | 深度求索（DeepSeek） | **极致性价比与推理之王**：训练/推理成本极低、数学/代码顶级，开源生态强势。 | 深度开源 |
| <img src="./qwen.svg" width="32"> | **Qwen**（通义千问） | Qwen2.5-Max, Qwen3-235B-A22B, QwQ-32B, QvQ, Qwen-VL-Max（VLM）, qwen-vl-ocr | 阿里巴巴（阿里云） | **开源半壁江山**：衍生模型数量全球第一，多语言、Agent 工具调用强；VLM 与 VLM-OCR 复用同套接口。 | 深度开源 |
| <img src="./doubao.svg" width="32"> | **Doubao**（豆包） | Doubao-1.5-pro, Doubao-pro-32k | 字节跳动（火山引擎） | **C 端霸主**：国内 C 端用户量极大，稀疏 MoE 架构降本提速。 | 闭源（提供 API） |
| <img src="./seed.svg" width="32"> | **Seed**（字节研究） | Seed-OSS-36B-Instruct | 字节跳动 Seed 团队 | **字节开源研究分支**：与 Doubao 商业线分离，专注开源前沿模型。 | 部分开源 |
| <img src="./claude.svg" width="32"> | **Claude** | Claude 3.5/3.7 Sonnet, Claude Opus 4 | Anthropic | **代码与长文本王者**：代码能力极强、200K 长上下文、安全对齐顶尖。 | 闭源（提供 API） |
| <img src="./gpt.svg" width="32"> | **GPT** | GPT-4o, o1/o3 系列, GPT-5 | OpenAI | **综合霸主**：生态最成熟，o 系列强推理 + 4o 多模态。 | 闭源（提供 API） |
| <img src="./gemini.svg" width="32"> | **Gemini** | Gemini 2.5 Pro, Gemini 2.5 Flash, Gemini 3 | Google DeepMind | **原生多模态 + 超长上下文**：百万级窗口，与 Google 生态深度绑定。 | 闭源（提供 API） |
| <img src="./zai.svg" width="32"> | **GLM**（智谱清言） | GLM-4-Plus, GLM-4-Flash, GLM-4.5, GLM-5 | 智谱 AI（Zhipu） | **全能六边形战士**：长程任务和代码均衡。 | 部分开源 |
| <img src="./kimi.svg" width="32"> | **Kimi**（Moonshot） | Kimi K1.5, Kimi K2 | 月之暗面（Moonshot） | **长文本与代码天花板**：超长上下文、Agent 能力强。 | 部分开源 |
| <img src="./llama.svg" width="32"> | **LLaMA** | Llama 3.1, Llama 3.2, Llama 4 | Meta | **开源生态基石**：全球开源社区底座，衍生模型无数。 | 完全开源 |
| <img src="./hunyuan.svg" width="32"> | **Hunyuan**（混元） | Hunyuan-Large, Hunyuan-Turbo, Hunyuan-A13B-Instruct, Hunyuan-MT | 腾讯 | **多模态与长文本专家**：深度整合微信生态。 | 部分开源 |
| <img src="./grok.svg" width="32"> | **Grok** | Grok-3, Grok-4 | xAI | **实时联网狂魔**：深度绑定 X 平台，数学/代码强。 | 闭源（提供 API） |
| <img src="./commanda.svg" width="32"> | **Command** | Command-R, Command-R+ | Cohere | **企业级 RAG 专家**：受监管行业首选。 | 闭源（提供 API） |
| <img src="./mistral.svg" width="32"> | **Mistral** | Mistral Large 2, Mistral Medium, Codestral, Pixtral, Magistral | Mistral AI | **欧洲开源先锋**：高性价比、低延迟、企业合规友好。 | 部分开源 |
| <img src="./minimax-color.svg" width="32"> | **MiniMax**（海螺） | MiniMax-M1, MiniMax-M2.x | MiniMax | **多模态原生**：文本/音频/图像/视频全模态。 | 闭源（提供 API） |
| <img src="./step.svg" width="32"> | **Step**（阶跃） | Step-2, Step-3.5-Flash | 阶跃星辰（StepFun） | **多模态新锐**：强语音/视觉/视频多模态。 | 闭源（提供 API） |
| <img src="./ling.svg" width="32"> | **Ling / Ring** | Ling-mini-2.0, Ling-flash-2.0, Ling-Plus, Ring-flash-2.0, Ring-Lite-V1 | InclusionAI（蚂蚁百灵） | **蚂蚁开源系列**：MoE 架构，Ring 主打推理增强。 | 完全开源 |
| <img src="./ernie.svg" width="32"> | **ERNIE**（文心一言） | ERNIE 4.0, ERNIE 4.5, ERNIE-4.5-Turbo, ERNIE-Speed-Pro | 百度 | **中文知识增强**：金融/政企落地广泛。 | 闭源（提供 API） |
| <img src="./pangu.svg" width="32"> | **PanGu**（盘古） | PanGu-Pro-72B, PanGu-Mini, 盘古行业大模型 | 华为 | **工业界老炮**：政企、工业制造、气象等硬核 B 端。 | 闭源（提供 API） |
| <img src="./TeleAI.svg" width="32"> | **TeleChat** | TeleChat-12B, TeleChat2-115B | 中国电信 TeleAI | **运营商大模型**：政企与运营商场景。 | 部分开源 |
| <img src="./spark.svg" width="32"> | **Spark**（讯飞星火） | Spark X1, Spark 4.0 | 科大讯飞 | **教育与医疗落地王**：软硬结合做得最好。 | 闭源（提供 API） |
| <img src="./sensenova.svg" width="32"> | **SenseChat / 日日新** | SenseChat 5.5, SenseNova | 商汤科技 | **视觉与多模态老将**：图文融合理解出色。 | 闭源（提供 API） |
| <img src="./baichuan.svg" width="32"> | **Baichuan**（百川） | Baichuan4-Air, Baichuan3-Turbo | 百川智能 | **MoE 架构探索者**：企业场景专项优化。 | 部分开源 |
| <img src="./yi.svg" width="32"> | **Yi**（零一万物） | Yi-Lightning, Yi-VL | 零一万物 | **代码与开源贡献活跃**：Yi-VL 多模态见长。 | 部分开源 |

---

## 二、OCR（ocr）

> 仅传统 OCR（非 VLM 系）。独立 endpoint `/v1/ocr`，按页计费。VLM 系 OCR（qwen-vl-ocr / deepseek-ocr）归在「对话」章节。

| Logo | 模型系列 | 代表版本 | 厂商 | 核心标签与亮点 | 开源状态 |
|---|---|---|---|---|---|
| <img src="./paddle.svg" width="32"> | **PaddleOCR** | PaddleOCR v3/v4, Paddle Table | PaddlePaddle（百度） | **中英文 OCR 标准方案**：开源、轻量、表格/版面识别都有。 | 完全开源 |

---

## 三、文生图（image）

> Endpoint `/v1/images/generations`，按张计费。

| Logo | 模型系列 | 代表版本 | 厂商 | 核心标签与亮点 | 开源状态 |
|---|---|---|---|---|---|
| <img src="./qwen.svg" width="32"> | **Wan**（通义万相） | Wan2.7-Image, qwen-image-2-0, qwen-image-edit | 阿里巴巴（与 Qwen 同厂不同系列） | **阿里视觉创作主力**：与文本 Qwen 分开运营，共用 logo。 | 部分开源 |
| <img src="./qwen.svg" width="32"> | **Z-Image**（通义万相 MAI） | Z-Image, Z-Image-Turbo | 阿里通义 MAI 团队 | **快速文生图**：主打实时/批量出图，秒级生成。共用 Qwen logo。 | 部分开源 |
| <img src="./hunyuan.svg" width="32"> | **Hunyuan-DiT** | Hunyuan-DiT | 腾讯 | **DiT 文生图开源**：腾讯混元图像生成，已开源。共用 Hunyuan logo。 | 部分开源 |
| <img src="./stability.svg" width="32"> | **Stable Diffusion** | SDXL, SD 3, FLUX | Stability AI / Black Forest Labs | **开源文生图标准**：社区生态最丰富，LoRA 微调主力载体。 | 完全开源 |
| <img src="./midjourney.svg" width="32"> | **Midjourney** | v6, v7 | Midjourney Inc. | **商业文生图天花板**：审美主导，Discord 起家。 | 闭源（提供 API） |

---

## 四、文生视频（video）

> Endpoint `/v1/videos/generations`，按秒计费。响应较慢，建议异步任务方式调用。

| Logo | 模型系列 | 代表版本 | 厂商 | 核心标签与亮点 | 开源状态 |
|---|---|---|---|---|---|
| <img src="./qwen.svg" width="32"> | **Wan**（通义万相） | Wan2.2-I2V-A14B, Wan2.2-T2V-A14B | 阿里巴巴 | **图/文驱动生视频**：高分辨率短视频，运动连贯性优秀。共用 Qwen logo。 | 部分开源 |
| <img src="./kolors.svg" width="32"> | **Kolors / Kling**（可灵） | Kolors, Kling | 快手 | **国产视频生成黑马**：效果对标 Sora/Veo，国内 C 端火爆。 | 部分开源 |

---

## 五、文转音 TTS（tts）

> Endpoint `/v1/audio/speech`，按万字符计费，返回音频流。

| Logo | 模型系列 | 代表版本 | 厂商 | 核心标签与亮点 | 开源状态 |
|---|---|---|---|---|---|
| <img src="./minimax-color.svg" width="32"> | **MiniMax Speech** | speech-02-hd, speech-2.8-turbo | MiniMax | **AI 语音合成**：与 Hailuo 视频同源。共用 MiniMax logo。 | 闭源（提供 API） |
| <img src="./qwen.svg" width="32"> | **CosyVoice / Sambert** | CosyVoice v1/v2, Sambert 系列 | 阿里通义实验室 | **国产 TTS 标杆**：零样本声音克隆 + 多说话人风格。共用 Qwen logo。 | 部分开源 |
| <img src="./f5-tts.svg" width="32"> | **F5-TTS** | F5-TTS | 上海交大 | **学术开源 TTS**：论文驱动的零样本 TTS。 | 完全开源 |
| <img src="./kokoro.svg" width="32"> | **Kokoro / kokoro-dt** | Kokoro-82M | 社区开源 | **轻量级开源 TTS**：82M 参数高质量多语言合成，端侧部署友好。 | 完全开源 |

---

## 六、音转文 ASR（asr）

> Endpoint `/v1/audio/transcriptions`，按分钟计费，multipart 上传音频。

| Logo | 模型系列 | 代表版本 | 厂商 | 核心标签与亮点 | 开源状态 |
|---|---|---|---|---|---|
| <img src="./TeleAI.svg" width="32"> | **TeleSpeechASR** | TeleSpeechASR | 中国电信 TeleAI | **中文 ASR**：呼叫中心/会议转写场景优化。共用 TeleAI logo。 | 部分开源 |
| <img src="./qwen.svg" width="32"> | **Qwen-ASR** | qwen3-asr-flash, qwen3-asr-flash-realtime | 阿里通义 | **多语种 ASR**：含实时转写版本。共用 Qwen logo。 | 闭源（提供 API） |

---

## 七、嵌入（embed）

> Endpoint `/v1/embeddings`，按 token 计费。

| Logo | 模型系列 | 代表版本 | 厂商 | 核心标签与亮点 | 开源状态 |
|---|---|---|---|---|---|
| <img src="./baai.svg" width="32"> | **BGE / BAAI** | BGE-Large-Zh, BGE-M3, bge-base-en-v1.5 | 智源研究院（BAAI） | **中文 RAG 默认选择**：开源、轻量、性能稳定。 | 完全开源 |
| <img src="./bce.svg" width="32"> | **BCE / 有道** | bce-embedding-base-v1 | 网易有道 | **垂直领域优化**：QAnything 框架配套，中英双语优化。 | 完全开源 |
| <img src="./qwen.svg" width="32"> | **GTE** | gte-large, gte-base | 阿里通义 | **多场景文本嵌入**：支持稠密/稀疏混合检索。共用 Qwen logo。 | 部分开源 |
| <img src="./qwen.svg" width="32"> | **text-embedding-v3**（通义） | text-embedding-v1/v2/v3, text-embedding-async-v1, qwen3-embedding-0.6b/4b/8b | 阿里通义 | **国产替代首选**：兼容 OpenAI embeddings 协议，多档 API。共用 Qwen logo。 | 闭源（提供 API） |
| <img src="./gpt.svg" width="32"> | **text-embedding-3** | text-embedding-3-large, text-embedding-3-small | OpenAI | **海外通用向量基础**：3072 维稠密向量。 | 闭源（提供 API） |
| <img src="./commanda.svg" width="32"> | **Cohere Embed** | embed-multilingual-v3 | Cohere | **多语言企业向量**：100+ 语言支持。共用 Cohere logo。 | 闭源（提供 API） |

---

## 八、重排序（rerank）

> Endpoint `/v1/rerank`，按 token 计费。RAG 流程提升答案准确率的关键一环。

| Logo | 模型系列 | 代表版本 | 厂商 | 核心标签与亮点 | 开源状态 |
|---|---|---|---|---|---|
| <img src="./baai.svg" width="32"> | **BGE Reranker** | BGE-Reranker-V2-M3, bge-reranker-large | 智源研究院（BAAI） | **中文重排标杆**：中文 RAG 默认选择，与 BGE Embedding 配套使用。 | 完全开源 |
| <img src="./bce.svg" width="32"> | **BCE Reranker** | bce-reranker-base-v1 | 网易有道 | **垂直领域优化**：QAnything 框架配套，与 BCE Embedding 配套。 | 完全开源 |
| <img src="./qwen.svg" width="32"> | **Qwen3 Reranker** | qwen3-reranker-0.6b, qwen3-reranker-4b, qwen3-reranker-8b, qwen3-vl-reranker-8b | 阿里通义 | **大参数重排**：覆盖 0.6B/4B/8B 档位 + 视觉重排。共用 Qwen logo。 | 部分开源 |

