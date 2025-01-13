export function initializeFileUpload() {
    const dropzone = document.getElementById('dropzone');
    const fileInput = document.getElementById('file-upload');
    const textarea = document.getElementById('content');
    const toggleBtn = document.getElementById('toggle-input');
    const filenameInput = document.getElementById('filename');

    if (!dropzone || !fileInput || !textarea || !toggleBtn) {
        return;
    }

    function toggleInput() {
        const isFileMode = textarea.style.display === 'none';
        textarea.style.display = isFileMode ? 'block' : 'none';
        dropzone.style.display = isFileMode ? 'none' : 'flex';
        toggleBtn.textContent = isFileMode ? 'Switch to File Upload' : 'Switch to Text Input';
        
        // Clear the other input when switching
        if (isFileMode) {
            fileInput.value = '';
            clearPreview();
            if (filenameInput) {
                filenameInput.value = '';
            }
        } else {
            textarea.value = '';
        }
    }

    function handleFileSelect(file) {
        if (!file) return;
    
        const previewContainer = document.getElementById('preview-container');
        const imagePreview = document.getElementById('image-preview');
        const dropzoneText = document.querySelector('.dropzone-text');
    
        // Clear any existing preview
        clearPreview();
    
        // Update filename input if it exists
        if (filenameInput && file.name) {
            filenameInput.value = file.name;
        }
    
        // If it's an image, show the preview
        if (file.type.startsWith('image/')) {
            const reader = new FileReader();
            reader.onload = function(e) {
                imagePreview.src = e.target.result;
                previewContainer.style.display = 'block';
                dropzoneText.style.display = 'none';
            };
            reader.readAsDataURL(file);
        }
    }

    function clearPreview() {
        const previewContainer = document.getElementById('preview-container');
        const imagePreview = document.getElementById('image-preview');
        const dropzoneText = document.querySelector('.dropzone-text');
        
        imagePreview.src = '';
        previewContainer.style.display = 'none';
        dropzoneText.style.display = 'block';
    }

    // Toggle button click handler
    toggleBtn.addEventListener('click', toggleInput);

    // Drag and drop handlers
    dropzone.addEventListener('dragover', (e) => {
        e.preventDefault();
        e.stopPropagation();
        dropzone.classList.add('dragover');
    });

    dropzone.addEventListener('dragleave', (e) => {
        e.preventDefault();
        e.stopPropagation();
        dropzone.classList.remove('dragover');
    });

    dropzone.addEventListener('drop', (e) => {
        e.preventDefault();
        e.stopPropagation();
        dropzone.classList.remove('dragover');
        
        const file = e.dataTransfer.files[0];
        if (file) {
            const dataTransfer = new DataTransfer();
            dataTransfer.items.add(file);
            fileInput.files = dataTransfer.files;
            handleFileSelect(file);
        }
    });

    // File input change handler
    fileInput.addEventListener('change', (e) => {
        const file = e.target.files[0];
        if (file) {
            handleFileSelect(file);
        }
    });

    // Click handler for the dropzone
    dropzone.addEventListener('click', () => {
        fileInput.click();
    });

    // Paste handler for images
    document.addEventListener('paste', (e) => {
        const items = e.clipboardData.items;
        for (let i = 0; i < items.length; i++) {
            if (items[i].type.indexOf('image') !== -1) {
                const file = items[i].getAsFile();
                const dataTransfer = new DataTransfer();
                dataTransfer.items.add(file);
                fileInput.files = dataTransfer.files;
                handleFileSelect(file);
                if (textarea.style.display !== 'none') {
                    toggleInput();
                }
                break;
            }
        }
    });
} 