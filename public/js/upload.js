export function initializeFileUpload() {
    const form = document.getElementById('paste-form');
    const toggleBtn = document.getElementById('toggle-input');
    const textarea = document.getElementById('content');
    const dropzone = document.getElementById('dropzone');
    const fileInput = document.getElementById('file-upload');
    const filenameInput = document.getElementById('filename');

    if (!form || !toggleBtn || !textarea || !dropzone || !fileInput) return;

    let isFileMode = false;

    // Toggle between text and file input modes
    toggleBtn.addEventListener('click', () => {
        isFileMode = !isFileMode;
        
        if (isFileMode) {
            textarea.style.display = 'none';
            dropzone.style.display = 'block';
            toggleBtn.textContent = 'Switch to Text Input';
            textarea.value = '';
        } else {
            textarea.style.display = 'block';
            dropzone.style.display = 'none';
            toggleBtn.textContent = 'Switch to File Upload';
            fileInput.value = '';
        }
    });

    // Handle file selection
    fileInput.addEventListener('change', handleFileSelect);

    // Handle drag and drop events
    dropzone.addEventListener('dragenter', preventDefault);
    dropzone.addEventListener('dragover', preventDefault);
    dropzone.addEventListener('dragleave', handleDragLeave);
    dropzone.addEventListener('drop', handleDrop);

    // Handle click to upload
    dropzone.addEventListener('click', () => {
        fileInput.click();
    });

    // Handle paste events
    document.addEventListener('paste', handlePaste);

    function preventDefault(e) {
        e.preventDefault();
        e.stopPropagation();
        dropzone.classList.add('dragover');
    }

    function handleDragLeave(e) {
        e.preventDefault();
        e.stopPropagation();
        dropzone.classList.remove('dragover');
    }

    function handleDrop(e) {
        e.preventDefault();
        e.stopPropagation();
        dropzone.classList.remove('dragover');

        const dt = e.dataTransfer;
        if (dt.files && dt.files.length > 0) {
            handleFiles(dt.files);
        }
    }

    function handlePaste(e) {
        if (!isFileMode) return;

        const items = (e.clipboardData || e.originalEvent.clipboardData).items;
        for (const item of items) {
            if (item.kind === 'file') {
                const file = item.getAsFile();
                handleFiles([file]);
                break;
            }
        }
    }

    function handleFileSelect(e) {
        if (e.target.files && e.target.files.length > 0) {
            handleFiles(e.target.files);
        }
    }

    function handleFiles(files) {
        if (files.length > 0) {
            const file = files[0];
            if (filenameInput) {
                filenameInput.value = file.name;
            }
            // Create a new FileList containing only the first file
            const dataTransfer = new DataTransfer();
            dataTransfer.items.add(file);
            fileInput.files = dataTransfer.files;
        }
    }

    // Form validation
    form.addEventListener('submit', (e) => {
        if (isFileMode && (!fileInput.files || fileInput.files.length === 0)) {
            e.preventDefault();
            alert('Please select a file to upload');
        } else if (!isFileMode && !textarea.value.trim()) {
            e.preventDefault();
            alert('Please enter some text content');
        }
    });
} 