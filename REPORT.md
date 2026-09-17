# Scribblebox 報告

**專案**：Scribblebox — 一個 Scribblenauts 風格的簡易解謎遊戲
**原始碼**：https://github.com/meowings20000/scribblebox（`main` 分支）
**本機執行**：`docker compose up -d --build` 後開 http://localhost:3003
**日期**：2026-09-16

---

## 1. 摘要

Scribblebox 是一個可以在瀏覽器直接玩的解謎遊戲：**在筆記本輸入物件名稱，物件就會出現在 2D 世界裡**，而物件有真正的性質（可燃、可爬、可切、可浮、可導電、爆炸、容器……），用它們去達成每道題的目標。核心是「一個輸入框就是整個遊戲的介面」，以及**每道題都有很多種解法**。

完成的三塊功能：

1. **遊戲本體** — 284 個名詞的字典、51 個修飾詞、規則式物理（重力堆疊、浮沉、結冰、火與水、切割、解鎖、爆炸、通電、攀爬）、四道刻意簡單的題目、全部用頁面自己畫的 inline SVG 呈現。
2. **四選一提示** — 按一下就得到四個物件，**其中恰好一個真的能過關**，而且這不是標籤而是引擎用模擬證明過的性質。
3. **可選 AI（自備 key 與 URL）** — 由模型判對錯、生成新題目、提出四個提示；同時有一個設定面板讓使用者填入自己的 base URL、key 與 model。

全程離線可玩、零外部素材、零相依套件，AI 完全可選：沒有填 key 的時候，遊戲功能一個都不缺。

---

## 2. 需求對照表

使用者每一句要求 → 實際交付 → 用什麼證明。

| 使用者的要求（原文） | 交付內容 | 驗證方式 |
|---|---|---|
| 「你知道 scribblenauts 嗎 我們做個簡單版的」 | 打字生成物件 + 規則式物理 + 四道多解題；物件是頁面自繪 SVG | 真棧 smoke：四題都以最直覺的物件解開（ladder／bridge／match／key），53 項通過 |
| 「可以接 ai 的 讓 ai 判斷對錯和生題」 | `POST /api/ai/judge`（判對錯）、`POST /api/ai/puzzles`（生題）、`POST /api/ai/hint`（AI 版四選一） | 用假的 OpenAI 相容端點跑通全流程（`backend/ai` 14 個測試 + `backend/api` 的 AI 測試）；AI 生的題目一律先經引擎驗證 |
| 「記得要做一個位置讓他們填入 key 和 url」 | 側欄 **AI features (optional)** 面板：base URL、key（密碼框，可切換顯示）、model、Save／Forgot、**Test connection** | 前端 252 項檢查 + 5 個 mock 的 Playwright 測試；測試斷言 key 不出現在回應、錯誤訊息或 `console` |
| 「其實可以 svg 畫圖」 | 世界全部是 inline SVG：20 種造型共用一組圖形，粗細一致、平塗色、確定性渲染（禁用 `Math.random`／`Date.now`） | 前端測試逐一驗證 20 種造型都存在；瀏覽器測試以 `g[data-key="ladder"]` 這類 DOM 斷言取代像素比對 |
| 「我説小學生程度是指題目難度」「不要太難」 | 地圖 20×10、目標距玩家 3–8 格（硬上限 12 格）、河流 2–4 格寬、每題 ≤3 次生成、≤25 tick 可解、玩家移動 ≤12 格；摔不死、不計時 | 世界套件的易度測試 + AI 生題的易度驗證（不合格會回報「too hard for the game's difficulty」） |
| 「也可以按按鈕提示 那就可以 4 選 1（只有一個對）」 | `POST /api/worlds/{id}/hint` 回傳四個選項，**恰好一個能解**；正解不標示；按下時再走 `hint/choose` 重新驗證落點 | `TestHintOffersFourVerifiedOptionsForEveryPuzzle`（逐題模擬四個選項）、`TestTheFourChoiceHintIsVerifiedEndToEnd`（走 HTTP 同樣驗證）、`TestChoosingAHintOptionIsVerifiedWhenItIsPressed`（在給出與按下之間改變世界） |

---

## 3. 系統架構

```
browser  index.html + app.js（vanilla，無框架、無 build step）
   │  same-origin /api/...（Docker 內由 Nginx 轉發）
   ▼
backend/api      HTTP 層：worlds、hints、AI proxy、CORS、長度與範圍限制、嚴格 JSON
backend/world    網格與地形、十條物理規則、玩家、四道題、提示的驗證與沙盒
backend/lexicon  284 個名詞、51 個修飾詞、即興物件、自動完成
backend/ai       OpenAI 相容客戶端、裁判、生題、提示建議
```

**為什麼這樣切**：文法與性質判定（哪些字可燃、可爬、可導電）是所有玩法共用的知識，集中在 `lexicon`；世界規則（物件之間怎麼互相影響）集中在 `world`；`api` 只做輸入輸出與邊界；`ai` 是最外層、可抽換的建議來源。

關鍵設計決定：

- **引擎是確定性的規則系統，不是模型驅動**：同一個輸入永遠得到同一個結果，這讓「提示只有一個對」「AI 生的題目可以被拒絕」這類承諾可以用測試釘住。
- **AI 只能提議，引擎才有裁決權**：模型給的題目、提示候選、對錯判斷，全部先經過引擎檢查。
- **憑證只在請求中出現**：key 由玩家瀏覽器每次帶上，只為該次呼叫使用，伺服器不落地、不記錄。
- **邊界都寫死並測試**：body 上限 64 KiB、片語 ≤60 字、一次 ≤40 tick、worlds 上限 500 個（淘汰最舊）、非 GET 變更一律 POST、Origin 白名單。

主要 API：

| Method | Path | 用途 |
|---|---|---|
| GET | `/api/health`、`/api/puzzles`、`/api/words?q=` | 狀態、四道題（**絕不含答案**）、自動完成＋造型 |
| POST | `/api/worlds` | `{"puzzleId":…}` 或 `{"aiSpec":…}` |
| GET | `/api/worlds/{id}` | 畫面渲染用的完整狀態 |
| POST | `/api/worlds/{id}/spawn｜step｜act｜reset` | 生成物件、推進時間、移動與拿放、重置 |
| POST | `/api/worlds/{id}/hint`、`…/hint/choose` | 四個已驗證選項；按下選擇時重新驗證落點 |
| POST | `/api/ai/judge｜puzzles｜hint｜ping` | 判對錯、生題、AI 提示、測試連線 |

---

## 4. 核心設計：四選一提示為什麼可信

「四個裡面只有一個對」如果只是文案，對玩家毫無價值。這裡它是一條**被引擎證明過的性質**：

1. **每個候選都先在沙盒複本裡試跑**。`world` 會在世界的副本中生成該物件、推進時間、看目標是否達成；能解開的留作正解，確定解不開的才當干擾項。任何一個干擾項若意外能解，就丟掉換下一個。
2. **湊不出四個已驗證選項就拒絕回答**，而不是提供可能錯誤的提示。
3. **正解不出現在任何地方**：回傳只有字串選項，位置由世界 id 與狀態決定性輪替；測試會走訪整份 JSON，確認沒有名為 `correct`／`solution`／`answer` 的欄位，畫面上也沒有任何標記。
4. **按下時重新驗證**。這是這一次最關鍵的修正：選項是在「被提供的那一刻」驗證的，但玩家會繼續玩——走位、堆東西、讓火焰蔓延——原本通過驗證的落點可能已經不能用。因此按下選擇改走 `POST /api/worlds/{id}/hint/choose`，由伺服器針對**當下的世界**重新取得一個已驗證落點、把世界跑到與驗證相同的穩定窗口（40 tick）後才回傳，所以承諾在玩家按下的那一刻仍然成立，而且結果立刻顯示在畫面上。
5. **落點會自己找地方**：原始的落點被堆滿時，引擎會沿著周圍逐步擴散的候選格尋找可驗證的位置；真的都不可行時，畫面會說明原因、清掉已經失效的選項，並指向 Reset。

---

## 5. AI 整合（可選、自備端點）

| 功能 | 端點 | 邊界 |
|---|---|---|
| AI 判對錯 | `POST /api/ai/judge` | 玩家用自己的話描述想法（含題目、已造物件與其性質、最近事件），模型回 `{approved, reason, confidence}`；通過時由引擎記錄為「外部裁判接受」並反映在目標狀態。判定不通過則世界完全不動 |
| AI 生題 | `POST /api/ai/puzzles` | 依 theme 與四種目標類型之一生成題目；**引擎先驗證才讓玩家玩**：地形代碼合法、`len(terrain)==w*h`、玩家起點不在牆內、目標物件存在、目標距玩家 12 格內、實體 ≤25、水域 ≤4 格寬。不合格回 `the AI's puzzle was rejected: …`，不會創建任何東西 |
| AI 四個提示 | `POST /api/ai/hint` | 模型提出四個物件，引擎逐一模擬後才回傳，仍然恰好一個真的能用；湊不出來就誠實報錯 |
| 測試連線 | `POST /api/ai/ping` | 先證明設定可用，回報模型與延遲；或誠實回報 401／逾時／連不上 |

**key 與 URL 的存放**：只存在玩家瀏覽器的一個 `localStorage` 項目（`scribblebox.ai`，全站唯一一處）。它只為該次請求送到自己的本機服務，伺服器**不寫入磁碟、不記錄、不回傳**；沒有按 Save 並主動要求之前不會送到任何地方，按 Forgot settings 即清除。若後端跑在 Docker 內，指向本機 AI 服務（例如 3000 埠的 new-api）要填 `http://host.docker.internal:3000/v1`；用 `go run` 跑後端則填 `http://127.0.0.1:3000/v1`。

**失敗一律誠實**：401/403 → 「the AI endpoint refused the key (401)」；連不上 → 「could not reach」；逾時 → 504；回傳不是 JSON → 502；並有測試斷言 key 不會出現在任何回應或錯誤訊息中。沒有填設定時，AI 按鈕停用並說明原因，遊戲其他部分完全不受影響。

**被拒絕的題目會得到一次修正機會**：引擎知道錯在哪，所以會把自己的錯誤訊息直接交回模型，要求它只修那個問題並重送整份 spec。實測有效——錄影裡 AI 生成的那道題就是這樣來的（回傳 `"repaired": true`）。提示詞也同步要求「世界盡量小（12×8）」，因為地形陣列越短，模型越容易算對、也越快回來。若第二次仍不可玩，會把原因告訴玩家，而不是給一道壞題。

---

## 6. 美術與難度

**美術**：世界完全由頁面自己畫的 inline SVG 構成，20 種造型（box、ladder、rope、plank、blob、circle、star、tree、flame、key、tool、animal、person、bottle、book、vehicle、chest、candle、flag、machine）共用一組線寬與平塗色，未知造型一律退回 `blob`。沒有圖片、字型、圖示庫或 CDN。渲染是確定性的：同一份狀態永遠畫出同一張圖（這也是測試能用 DOM 斷言取代像素比對的原因）。四選一提示的每個選項會用 `/api/words` 回傳的造型與顏色畫成它將在世界中出現的樣子。

**難度（這是使用者的重點）**：每道題都被強制在「小學生可以自己解開」的範圍內——地圖 20×10、目標距玩家 3–8 格、河流 2–4 格寬、每題最多 3 次生成與 25 tick 內可解、玩家移動不超過 12 格、不會摔死、不計時、Reset 永遠回到可解狀態。同一組限制也套用在 AI 生成的題目上（超過就拒絕並說明原因），所以「AI 生的題不會突然變得很難」。

---

## 7. 工程過程：哪些是並行完成、哪些是我親自修的

字典、世界引擎、前端三塊由三個並行的子代理實作，但**介面由我先釘死**（`Object` 結構與固定的 tag 詞彙表、`World` 的完整 API 與十條規則的執行順序、HTTP 合約與 DOM id）；AI 層、HTTP 層、整合、驗證與文件由我親自完成。契約先行是並行能成立的前提。

整合後我親自找出並修掉的問題（每一項都有對應的測試）：

1. **我自己的測試讓整個測試套件卡死**：逾時測試用了永不返回的 handler，導致 `httptest.Close()` 無止境等待；改成睡超過用戶端逾時再返回。
2. **提示需要第二個端點**（第 4 節）：從「提供時驗證」升級為「按下時重新驗證並穩定世界」。
3. **提示落點被堆滿就拒絕**：改為先試原始落點、再沿周圍擴散尋找，真的沒有才拒絕。
4. **贏了之後文字還在說沒達成**：選擇常常要一兩個 tick 後才生效（玩家爬上去），舊文字留在已通關的橫幅下方；現在會在目標達成的那一刻更正，並由兩個瀏覽器測試釘住。
5. **失效的選項還留在畫面上可按**：引擎無法背書時，現在會清掉舊選項並保留原因。
6. **四個選項縮圖長得一樣**：`/api/words` 現在回傳實際造型與顏色，選項各自畫成它將成為的圖示。
7. **我自己改 port 造成 CORS 擋掉自己**：前端移到 3003 但後端白名單還寫 3001，瀏覽器的同源請求被 403，遊戲整頁載不出來；修好並加上守門測試 `TestDefaultsAllowThePublishedFrontendPort`（讀 `docker-compose.yml` 比對白名單）。
8. **頁腳還說 canvas**：渲染器早已改成 SVG。
9. **AI 生成的題目算錯格子數**：接上真實端點後，模型給的 20×12 地形寫了 244 格（應為 240），引擎正確拒絕，但玩家只看到失敗。現在被拒絕的 spec 會得到一次修正機會——引擎把自己的錯誤訊息交回模型讓它自己修；錄影裡 AI 造的題就是這樣通過的（`"repaired": true`）。提示詞同時改成要求小世界（12×8），因為地形越短模型越容易算對。
10. **AI 的題目掛著內建題的名字**：目標面板原本用「目標類型」取名，於是 AI 造的燈塔題顯示成「Get the star out of the tree」。現在題目有自己的名字就用它——這也是錄影中看到的行為。

---

## 8. 實證

| 檢查 | 結果 |
|---|---|
| `go test -race -count=1 ./...` | **139 個測試全過**：lexicon 38、world 59、ai 16、api 22、main 4 |
| `npm test` | **252 項**前端與部署設定檢查通過 |
| `npm run test:e2e` | **20 個 Playwright 全過**，其中 2 個對真實 Docker 執行 |
| 真棧 smoke（純 HTTP 走完四題） | **53 項通過**；四題都以最直覺的物件解開（ladder／bridge／match／key） |
| 瀏覽器實測 | 四選一提示可在畫面上解題成功、回饋文字即時且誠實、零頁面錯誤 |
| 規模 | Go 約 12,500 行（含測試）、`app.js` 2,103 行、字典 284 名詞／51 修飾詞／20 造型／35 種性質、4 道題共 32 種已驗證解法 |
| 錄影 | `docs/scribblebox-demo.mp4`：**4 分 40 秒**真實遊玩 + 真實 AI 段（設定、測試連線、AI 四選一、判題、AI 生成並由引擎接受的題目） |
| 公開網址 | <https://scribblebox.meowmeow12245ouo.dpdns.org>（cloudflared tunnel `scribblebox` → 本機 3003；同一網域同時服務頁面與 API，53 項 smoke 對公開網址全過） |

值得單獨指出的測試：`TestHintOffersFourVerifiedOptionsForEveryPuzzle`（逐題把四個選項真的玩一遍，要求恰好一個贏）、`TestTheFourChoiceHintIsVerifiedEndToEnd`（改走 HTTP 再驗一次）、`TestChoosingAHintOptionIsVerifiedWhenItIsPressed`（在提供與按下之間刻意改變世界——這正是過去會失效的情境）、`TestHintMovesToANearbySpotWhenTheUsualOneIsBlocked`、`TestWordsReportsTheShapeOfAWordItKnows`、`TestDefaultsAllowThePublishedFrontendPort`。

---

## 9. 已知限制

- 字典有限（284 個名詞加上任意形容詞組合）；字典外的字會生成穩定、可玩的即興物件，但**不會自動擁有那個字該有的能力**。
- 物理是小規則引擎而非完整模擬：沒有動量、沒有液體流動、除了明文規則外沒有破壞效果。
- 四道內建題目是人工編寫與驗證的，刻意簡單；提示的干擾項也是人工挑選的明顯錯誤選項。
- AI 裁判的判斷**不由引擎複驗**（模型判的是概念，引擎判的是物理）；它的通過會被記錄為「外部裁判接受」，一般遊玩仍由引擎規則決定。
- 玩家若把某個落點堆滿，該處可能無法再提供已驗證的提示；引擎會說明並建議 Reset，而不是給一個不保證正確的提示。
- 玩家角色是簡單的攀爬者；題目配置刻意保持精簡（這也是地圖固定在 20×10 的原因）。

---

## 10. 下一步

- 增加題目數量與主題變化（例如「把動物送回籠子」「用水滅火」），維持同一組易度上限。
- 讓即興物件能依字尾或語意獲得有限能力（例如 `-er` 結尾傾向工具），讓字典外的字更「講理」。
- 把 AI 生的題目存成可重播的 spec（目前只在記憶體中），並加上「這題可解嗎」的引擎預檢。
- 記錄玩家的解法歷史，做成「用過幾種不同解法」的鼓勵式統計。

---

## 附錄：如何執行

```bash
docker compose up -d --build        # 前端 http://localhost:3003、後端 :8083（都只綁 127.0.0.1）
cd backend  && go test -race -count=1 ./...
cd frontend && npm test && APP_URL=http://localhost:3003 npm run test:e2e
```

檔案結構：

```
backend/{lexicon,world,ai,api}/   Go 服務：字典、世界引擎、AI 客戶端、HTTP 層
frontend/{index.html,app.js,styles.css,e2e.spec.js,frontend_test.js}
docs/{TECHNICAL_NOTE.md,DEMO_SCRIPT.md,shots/}
README.md
```
