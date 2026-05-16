export default {
    name: 'AIConfig',
    data() {
        return {
            loading: false,
            testing: false,
            form: {
                provider: 'openai_compatible',
                model: '',
                api_key: '',
                base_url: '',
                embed_provider: '',
                embed_model: '',
                embed_api_key: '',
                embed_base_url: '',
                system_prompt: '',
                style_preset: 'customer_service_formal',
                max_tokens: 500,
                temperature: 0.3,
                top_k: 4,
                retrieval_threshold: 0.3,
                guardrail_enabled: true,
                out_of_scope_message: 'Maaf, saya hanya bisa bantu seputar topik yang ada di knowledgebase kami.',
            },
            stylePresets: [
                { value: 'customer_service_formal', label: 'Customer Service (formal)' },
                { value: 'sales_casual', label: 'Sales (casual)' },
                { value: 'technical_support', label: 'Technical Support' },
                { value: 'custom', label: 'Custom (write your own prompt)' },
            ],
            testResult: null,
        }
    },
    methods: {
        async openModal() {
            $('#modalAIConfig').modal('show');
            await this.loadConfig();
        },
        async loadConfig() {
            this.loading = true;
            this.testResult = null;
            try {
                const resp = await window.http.get('/aireply/config');
                const r = resp.data.results;
                if (r) {
                    Object.assign(this.form, r);
                }
            } catch (e) {
                if (e.response && e.response.status !== 404) {
                    showErrorInfo(e.response?.data?.message || 'Failed to load config');
                }
            } finally {
                this.loading = false;
            }
        },
        async save() {
            this.loading = true;
            try {
                await window.http.put('/aireply/config', this.form);
                showSuccessInfo('Config saved');
            } catch (e) {
                showErrorInfo(e.response?.data?.message || 'Failed to save');
            } finally {
                this.loading = false;
            }
        },
        async testConnection() {
            this.testing = true;
            this.testResult = null;
            try {
                const resp = await window.http.post('/aireply/config/test');
                this.testResult = {
                    ok: true,
                    latency: resp.data.results.latency_ms,
                    body: resp.data.results.model_response,
                };
                showSuccessInfo('Provider OK');
            } catch (e) {
                this.testResult = { ok: false, body: e.response?.data?.message || e.message };
                showErrorInfo('Test failed: ' + this.testResult.body);
            } finally {
                this.testing = false;
            }
        },
    },
    template: `
    <div class="teal card" @click="openModal()" style="cursor: pointer">
        <div class="content">
            <a class="ui teal right ribbon label">AI</a>
            <div class="header">AI Configuration</div>
            <div class="description">
                Provider, model, style preset, guardrail & API keys
            </div>
        </div>
    </div>

    <div class="ui large modal" id="modalAIConfig">
        <i class="close icon"></i>
        <div class="header"><i class="cogs icon"></i> AI Auto-Reply Configuration</div>
        <div class="content">
            <div v-if="loading" class="ui active centered inline loader"></div>
            <form class="ui form" @submit.prevent="save">
                <div class="two fields">
                    <div class="field">
                        <label>Provider</label>
                        <select class="ui dropdown" v-model="form.provider">
                            <option value="openai_compatible">OpenAI-compatible (OpenRouter / Sumopod / DeepSeek / Groq / Ollama / OpenAI)</option>
                            <option value="anthropic">Anthropic (Claude)</option>
                        </select>
                    </div>
                    <div class="field">
                        <label>Model</label>
                        <input type="text" v-model="form.model" placeholder="e.g. claude-sonnet-4-6 or openai/gpt-4o-mini">
                    </div>
                </div>
                <div class="two fields">
                    <div class="field">
                        <label>API Key</label>
                        <input type="password" v-model="form.api_key" placeholder="sk-...">
                    </div>
                    <div class="field">
                        <label>Base URL (optional, openai-compatible only)</label>
                        <input type="text" v-model="form.base_url" placeholder="https://openrouter.ai/api/v1">
                    </div>
                </div>
                <div v-if="form.provider === 'anthropic'" class="ui yellow message">
                    <p>Anthropic does not provide embeddings. Configure an openai-compatible embed provider below.</p>
                </div>
                <div class="ui accordion" ref="advanced">
                    <div class="title"><i class="dropdown icon"></i> Embeddings (advanced)</div>
                    <div class="content">
                        <div class="two fields">
                            <div class="field">
                                <label>Embed Provider</label>
                                <select class="ui dropdown" v-model="form.embed_provider">
                                    <option value="">(use chat provider if openai-compatible)</option>
                                    <option value="openai_compatible">openai_compatible</option>
                                </select>
                            </div>
                            <div class="field">
                                <label>Embed Model</label>
                                <input type="text" v-model="form.embed_model" placeholder="text-embedding-3-small">
                            </div>
                        </div>
                        <div class="two fields">
                            <div class="field">
                                <label>Embed API Key (if different)</label>
                                <input type="password" v-model="form.embed_api_key">
                            </div>
                            <div class="field">
                                <label>Embed Base URL</label>
                                <input type="text" v-model="form.embed_base_url">
                            </div>
                        </div>
                    </div>
                </div>
                <div class="field">
                    <label>Style Preset</label>
                    <select class="ui dropdown" v-model="form.style_preset">
                        <option v-for="s in stylePresets" :key="s.value" :value="s.value">{{ s.label }}</option>
                    </select>
                </div>
                <div class="field" v-if="form.style_preset === 'custom'">
                    <label>Custom System Prompt</label>
                    <textarea rows="4" v-model="form.system_prompt"
                              placeholder="Write your own instructions: persona, language, tone, format..."></textarea>
                </div>
                <div class="four fields">
                    <div class="field">
                        <label>Max Tokens</label>
                        <input type="number" v-model.number="form.max_tokens" min="1" max="8000">
                    </div>
                    <div class="field">
                        <label>Temperature</label>
                        <input type="number" step="0.05" v-model.number="form.temperature" min="0" max="2">
                    </div>
                    <div class="field">
                        <label>Top K</label>
                        <input type="number" v-model.number="form.top_k" min="1" max="20">
                    </div>
                    <div class="field">
                        <label>Threshold</label>
                        <input type="number" step="0.05" v-model.number="form.retrieval_threshold" min="0" max="1">
                    </div>
                </div>
                <div class="field">
                    <div class="ui toggle checkbox">
                        <input type="checkbox" v-model="form.guardrail_enabled">
                        <label>Enable guardrail (refuse out-of-scope questions)</label>
                    </div>
                </div>
                <div class="field">
                    <label>Out-of-scope reply</label>
                    <input type="text" v-model="form.out_of_scope_message">
                </div>
                <div v-if="testResult" :class="['ui', testResult.ok ? 'green' : 'red', 'message']">
                    <div class="header">Test {{ testResult.ok ? 'OK' : 'failed' }}</div>
                    <p v-if="testResult.latency">Latency: {{ testResult.latency }}ms</p>
                    <p>{{ testResult.body }}</p>
                </div>
                <div class="ui buttons">
                    <button type="button" class="ui button" :class="{ loading: testing }" @click="testConnection">
                        <i class="plug icon"></i> Test Connection
                    </button>
                    <div class="or"></div>
                    <button type="submit" class="ui primary button" :class="{ loading: loading }">
                        <i class="save icon"></i> Save
                    </button>
                </div>
            </form>
        </div>
    </div>
    `,
}
