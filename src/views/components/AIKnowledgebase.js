export default {
    name: 'AIKnowledgebase',
    data() {
        return {
            loading: false,
            uploading: false,
            reindexing: false,
            file: null,
            docs: [],
            pollHandle: null,
        }
    },
    methods: {
        async openModal() {
            $('#modalAIKB').modal('show');
            await this.refresh();
            if (this.pollHandle) clearInterval(this.pollHandle);
            this.pollHandle = setInterval(() => this.refresh(true), 4000);
            $('#modalAIKB').modal({
                onHidden: () => {
                    if (this.pollHandle) clearInterval(this.pollHandle);
                    this.pollHandle = null;
                }
            });
        },
        async refresh(silent = false) {
            if (!silent) this.loading = true;
            try {
                const resp = await window.http.get('/aireply/documents');
                this.docs = resp.data.results || [];
            } catch (e) {
                if (!silent) showErrorInfo(e.response?.data?.message || 'Failed to load documents');
            } finally {
                if (!silent) this.loading = false;
            }
        },
        onFile(ev) {
            this.file = ev.target.files[0] || null;
        },
        async upload() {
            if (!this.file) { showErrorInfo('Pick a file first'); return; }
            this.uploading = true;
            try {
                const fd = new FormData();
                fd.append('file', this.file);
                const resp = await window.http.post('/aireply/documents', fd, {
                    headers: { 'Content-Type': 'multipart/form-data' }
                });
                showSuccessInfo('Uploaded — processing in background');
                this.file = null;
                if (this.$refs.fileInput) this.$refs.fileInput.value = '';
                await this.refresh();
            } catch (e) {
                showErrorInfo(e.response?.data?.message || 'Upload failed');
            } finally {
                this.uploading = false;
            }
        },
        async remove(id) {
            if (!confirm('Delete this document and all its chunks?')) return;
            try {
                await window.http.delete('/aireply/documents/' + encodeURIComponent(id));
                showSuccessInfo('Deleted');
                await this.refresh();
            } catch (e) {
                showErrorInfo(e.response?.data?.message || 'Delete failed');
            }
        },
        async reindex() {
            if (!confirm('Re-embed every chunk for this device? Used after changing the embed model.')) return;
            this.reindexing = true;
            try {
                await window.http.post('/aireply/documents/reindex');
                showSuccessInfo('Reindex complete');
                await this.refresh();
            } catch (e) {
                showErrorInfo(e.response?.data?.message || 'Reindex failed');
            } finally {
                this.reindexing = false;
            }
        },
        statusColor(s) {
            switch (s) {
                case 'ready': return 'green';
                case 'processing': return 'blue';
                case 'failed': return 'red';
                default: return 'grey';
            }
        },
        fmtSize(b) {
            if (b < 1024) return b + ' B';
            if (b < 1024 * 1024) return (b / 1024).toFixed(1) + ' KB';
            return (b / 1024 / 1024).toFixed(1) + ' MB';
        },
    },
    template: `
    <div class="teal card" @click="openModal()" style="cursor: pointer">
        <div class="content">
            <a class="ui teal right ribbon label">AI</a>
            <div class="header">Knowledgebase</div>
            <div class="description">
                Upload PDF / DOCX / TXT documents for RAG retrieval
            </div>
        </div>
    </div>

    <div class="ui large modal" id="modalAIKB">
        <i class="close icon"></i>
        <div class="header"><i class="book icon"></i> AI Knowledgebase</div>
        <div class="content">
            <div class="ui form">
                <div class="field">
                    <label>Upload document (PDF, DOCX, TXT, MD)</label>
                    <input type="file" ref="fileInput" accept=".pdf,.docx,.txt,.md" @change="onFile">
                </div>
                <button class="ui primary button" :class="{loading: uploading}" :disabled="!file" @click="upload">
                    <i class="upload icon"></i> Upload
                </button>
                <button class="ui button" :class="{loading: reindexing}" @click="reindex">
                    <i class="redo icon"></i> Re-index All
                </button>
                <button class="ui button" @click="refresh(false)"><i class="refresh icon"></i> Refresh</button>
            </div>
            <div class="ui divider"></div>
            <div v-if="loading" class="ui active centered inline loader"></div>
            <table v-else class="ui celled table">
                <thead>
                    <tr><th>Filename</th><th>Size</th><th>Chunks</th><th>Status</th><th>Created</th><th></th></tr>
                </thead>
                <tbody>
                    <tr v-if="!docs.length"><td colspan="6">No documents uploaded yet.</td></tr>
                    <tr v-for="d in docs" :key="d.id">
                        <td>{{ d.filename }}</td>
                        <td>{{ fmtSize(d.file_size) }}</td>
                        <td>{{ d.chunk_count }}</td>
                        <td>
                            <span :class="['ui', statusColor(d.status), 'label']">{{ d.status }}</span>
                            <div v-if="d.error_message" class="ui mini red text">{{ d.error_message }}</div>
                        </td>
                        <td>{{ new Date(d.created_at).toLocaleString() }}</td>
                        <td><button class="ui mini red button" @click="remove(d.id)"><i class="trash icon"></i></button></td>
                    </tr>
                </tbody>
            </table>
        </div>
    </div>
    `,
}
