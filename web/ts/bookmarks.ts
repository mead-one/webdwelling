document.addEventListener("DOMContentLoaded", function() {
    const expandAllFoldersButton: HTMLButtonElement | null = document.getElementById("expand-all-folders") as HTMLButtonElement;
    const collapseAllFoldersButton: HTMLButtonElement | null = document.getElementById("collapse-all-folders") as HTMLButtonElement;
    const bookmarkLIs: NodeListOf<HTMLLIElement> = document.querySelectorAll("li.bookmark");
    const folderLIs: NodeListOf<HTMLLIElement> = document.querySelectorAll("li.folder");
    const bookmarkButtons: NodeListOf<HTMLButtonElement> = document.querySelectorAll("button.add-bookmark");
    const editBookmarkButtons: NodeListOf<HTMLButtonElement> = document.querySelectorAll("button.edit-bookmark");
    const deleteBookmarkButtons: NodeListOf<HTMLButtonElement> = document.querySelectorAll("button.delete-bookmark");
    const folderButtons: NodeListOf<HTMLButtonElement> = document.querySelectorAll("button.add-folder");
    const renameFolderButtons: NodeListOf<HTMLButtonElement> = document.querySelectorAll("button.rename-folder");
    const deleteFolderButtons: NodeListOf<HTMLButtonElement> = document.querySelectorAll("button.delete-folder");
    const addBookmarkForm: HTMLFormElement | null = document.getElementById("add-bookmark-form") as HTMLFormElement;
    const editBookmarkForm: HTMLFormElement | null = document.getElementById("edit-bookmark-form") as HTMLFormElement;
    const addFolderForm: HTMLFormElement | null = document.getElementById("add-folder-form") as HTMLFormElement;
    const cancelAddBookmarkButton: HTMLButtonElement | null = document.getElementById("cancel-add-bookmark") as HTMLButtonElement;
    const cancelAddFolderButton: HTMLButtonElement | null = document.getElementById("cancel-add-folder") as HTMLButtonElement;

    if (expandAllFoldersButton !== null) expandAllFoldersButton.addEventListener("click", expandAllFolders);
    if (collapseAllFoldersButton !== null) collapseAllFoldersButton.addEventListener("click", collapseAllFolders);

    // Drag and drop
    bookmarkLIs.forEach(li => {
        li.setAttribute("draggable", "true");
        // Disable dragging of anchor element
        const link: HTMLAnchorElement | null = li.querySelector("a.bookmark-title") as HTMLAnchorElement;
        if (link !== null) link.draggable = false;

        li.addEventListener("dragstart", dragStartBookmark);
    });

    folderLIs.forEach(li => {
        li.setAttribute("draggable", "true");
        li.addEventListener("dragstart", dragStartFolder);
    });

    // folderLIs.forEach(li => {
    document.addEventListener("dragover", dragOverFolder);
    document.addEventListener("dragleave", dragLeaveFolder);
    document.addEventListener("drop", dropOnFolder);
    // });

    // Add bookmark buttons
    bookmarkButtons.forEach(button => {
        button.addEventListener("click", openAddBookmarkModal);
    });

    // Edit bookmark buttons
    editBookmarkButtons.forEach(button => {
        button.addEventListener("click", openEditBookmarkModal);
    });

    // Delete bookmark buttons
    deleteBookmarkButtons.forEach(button => {
        button.addEventListener("click", submitDeleteBookmark);
    });

    // Add folder buttons
    folderButtons.forEach(button => {
        button.addEventListener("click", openAddFolderModal);
    });

    // Rename folder buttons
    renameFolderButtons.forEach(button => {
        button.addEventListener("click", openRenameFolderForm);
    });

    // Delete folder buttons
    deleteFolderButtons.forEach(button => {
        button.addEventListener("click", submitDeleteFolder);
    });

    if (addBookmarkForm !== null) addBookmarkForm.addEventListener("submit", submitAddBookmarkForm);
    if (editBookmarkForm !== null) editBookmarkForm.addEventListener("submit", submitEditBookmarkForm);
    if (addFolderForm !== null) addFolderForm.addEventListener("submit", submitAddFolderForm);
    if (cancelAddBookmarkButton !== null) cancelAddBookmarkButton.addEventListener("click", closeAddBookmarkModal);
    if (cancelAddFolderButton !== null) cancelAddFolderButton.addEventListener("click", closeAddFolderModal);
});

type Bookmark = {
    ID: number;
    Title: string;
    URL: string;
    Tags: string;
    Public: boolean;
    FolderID: string | null;
    CreatedAt: string;
};

type BookmarkFolder = {
    ID: string;
    Name: string;
    Public: boolean;
    ParentFolderID: string | null;
    CreatedAt: string;
};

// Expand all folders
function expandAllFolders() {
    const folderDetails: NodeListOf<HTMLDetailsElement> = document.querySelectorAll("details.folder-details");
    folderDetails.forEach(d => {
        d.setAttribute("open", "");
    });
}

// Collapse all folders
function collapseAllFolders() {
    const folderDetails: NodeListOf<HTMLDetailsElement> = document.querySelectorAll("details.folder-details");
    folderDetails.forEach(d => {
        d.removeAttribute("open");
    });
}

// Drag start event for dragging a bookmark
function dragStartBookmark(event: DragEvent) {
    event.stopPropagation();
    if (event.target === null) {
        console.error("Target is null");
        return;
    }

    if (event.dataTransfer === null) {
        console.error("Data transfer is null");
        return;
    }

    const bookmarkLi: HTMLLIElement | null = event.target as HTMLLIElement;
    if (bookmarkLi === null) {
        console.error("Target is null");
        return;
    } else if (bookmarkLi.dataset.id === undefined) {
        console.error("Target dataset is undefined");
        return;
    }

    const bookmarkID: string | undefined = bookmarkLi.dataset.id;

    event.dataTransfer.setData("bookmark-id", bookmarkID);
    event.dataTransfer.setData("type", "bookmark");
}

function dragStartFolder(event: DragEvent) {
    event.stopPropagation();
    if (event.target === null) {
        console.error("Target is null");
        return;
    }

    if (event.dataTransfer === null) {
        console.error("Data transfer is null");
        return;
    }

    const folderLi: HTMLLIElement | null = event.target as HTMLLIElement;
    if (folderLi === null) {
        console.error("Target is null");
        return;
    } else if (folderLi.dataset.id === undefined) {
        console.error("Target dataset is undefined");
        return;
    }

    const folderID: string | undefined = folderLi.dataset.id;

    event.dataTransfer.setData("folder-id", folderID);
    event.dataTransfer.setData("type", "folder");
}

function dragOverFolder(event: DragEvent) {
    let draggedType: string | null, draggedLi: HTMLLIElement | null;
    event.preventDefault();

    if (event.dataTransfer === null) {
        console.error("Data transfer is null");
        return;
    }

    draggedType = event.dataTransfer.getData("type");

    switch (draggedType) {
    case "folder":
        const draggedFolderID: string | undefined = event.dataTransfer.getData("folder-id");
        draggedLi = document.getElementById(`folder-${draggedFolderID}`) as HTMLLIElement;
        break;
    case "bookmark":
        const draggedBookmarkID: string | undefined = event.dataTransfer.getData("bookmark-id");
        draggedLi = document.getElementById(`bookmark-${draggedBookmarkID}`) as HTMLLIElement;
        break;
    default:
        draggedLi = null;
    }

    const targetElement: HTMLElement | null = event.target as HTMLElement;

    let folderLi: HTMLLIElement | null;
    if (targetElement.classList.contains("folder")) {
        folderLi = targetElement as HTMLLIElement;
    } else if (targetElement.closest(".folder") !== null) {
        folderLi = targetElement.closest(".folder") as HTMLLIElement;
    } else {
        console.error("Target is not a folder");
        return;
    }

    if (folderLi === null) {
        console.error("Target is null");
        return;
    }

    if (draggedType === "bookmark" ||
        (draggedLi !== null && !(draggedLi.contains(targetElement)))) {
        folderLi.classList.add("dragover");
    }
}

function dragLeaveFolder(event: DragEvent) {
    event.preventDefault();
    const targetElement: HTMLElement | null = event.target as HTMLElement;

    let folderLi: HTMLLIElement | null;
    if (targetElement.classList.contains("folder")) {
        folderLi = targetElement as HTMLLIElement;
    } else if (targetElement.closest(".folder") !== null) {
        folderLi = targetElement.closest(".folder") as HTMLLIElement;
    } else {
        console.error("Target is null");
        return;
    }

    if (folderLi === null) {
        console.error("Target is null");
        return;
    }

    folderLi.classList.remove("dragover");
}

function dropOnFolder(event: DragEvent) {
    event.preventDefault();
    const targetElement: HTMLElement | null = event.target as HTMLElement;

    let folderLi: HTMLLIElement | null;
    if (targetElement.classList.contains("folder")) {
        folderLi = targetElement as HTMLLIElement;
    } else if (targetElement.closest(".folder") !== null) {
        folderLi = targetElement.closest(".folder") as HTMLLIElement;
    } else {
        console.error("Target is not a folder");
        return;
    }

    if (folderLi === null) {
        console.error("Target is null");
        return;
    }

    const folderID: string | undefined = folderLi.dataset.id;

    if (folderID === undefined) {
        console.error("Folder ID is undefined");
        return;
    }

    folderLi.classList.remove("dragover");

    if (event.dataTransfer === null) {
        console.error("Data transfer is null");
        return;
    } else if (!event.dataTransfer.types.includes("type")) {
        console.error("Unknown data type");
        return;
    }

    switch (event.dataTransfer.getData("type")) {
    case "bookmark":
        const bookmarkID: string | undefined = event.dataTransfer.getData("bookmark-id");

        if (bookmarkID === undefined) {
            console.error("Bookmark ID is undefined");
            return;
        }

        moveBookmarkToFolder(bookmarkID, folderID);
        break;
    case "folder":
        const draggedFolderID: string | undefined = event.dataTransfer.getData("folder-id");
        const draggedLi: HTMLLIElement | null = document.getElementById(`folder-${draggedFolderID}`) as HTMLLIElement;

        if (draggedFolderID === undefined) {
            console.error("Folder ID is undefined");
            return;
        } else if (draggedLi.contains(targetElement)) {
            console.error("Cannot move folder into itself or its descendants");
            return;
        }

        moveBookmarkFolderToFolder(draggedFolderID, folderID);
        break;
    default:
        console.error("Incompatible element dropped");
        return;
    }
}

// Open the add bookmark modal
function openAddBookmarkModal(event: MouseEvent) {
    const parentFolder: HTMLDivElement | null = (event.target as HTMLFormElement).closest(".folder");
    if (parentFolder === null) {
        console.error("No parent folder found");
        return;
    }

    const parentFolderID: string | undefined = parentFolder.dataset.id;
    const modalContainer: HTMLDivElement | null = document.getElementById("modal-container") as HTMLDivElement;
    const addBookmarkForm: HTMLFormElement | null = document.getElementById("add-bookmark-form") as HTMLFormElement;
    const folderSelectWrapper: HTMLDivElement | null = document.getElementById("add-bookmark-folder-select-wrapper") as HTMLDivElement;
    const folderSelect: HTMLDivElement | null = document.getElementById("folder-select") as HTMLDivElement;

    // Include the folder select sub-form
    folderSelectWrapper.appendChild(folderSelect);

    // Form elements
    const folderRadio: HTMLInputElement | null =
        addBookmarkForm.elements.namedItem("folder-select") as HTMLInputElement;
    const bookmarkTitle: HTMLInputElement =
        addBookmarkForm.elements.namedItem("title") as HTMLInputElement;

    if (folderSelectWrapper === null || addBookmarkForm === null || folderSelect === null ||
        modalContainer === null || folderRadio === null || bookmarkTitle === null) {
        console.error("Missing elements");
        return;
    }

    addBookmarkForm.reset();
    folderRadio.value = parentFolderID ? parentFolderID : "";
    modalContainer.style.display = "block";
    addBookmarkForm.style.display = "block";
    bookmarkTitle.focus();
}

function submitAddBookmarkForm(event: SubmitEvent) {
    event.preventDefault();
    const form: HTMLFormElement = event.target as HTMLFormElement;
    const titleInput: HTMLInputElement | null = form.elements.namedItem("title") as HTMLInputElement;
    const urlInput: HTMLInputElement | null = form.elements.namedItem("url") as HTMLInputElement;
    const tagsInput: HTMLInputElement | null = form.elements.namedItem("tags") as HTMLInputElement;
    const isPublicInput: HTMLInputElement | null = form.elements.namedItem("public") as HTMLInputElement;
    const folderRadio: HTMLInputElement | null =
        form.elements.namedItem("folder-select") as HTMLInputElement;

    const title = titleInput.value;
    const url = urlInput.value;
    const tags = tagsInput.value;
    const isPublic = isPublicInput.checked;
    const folderID = folderRadio.value === "0" ? null : folderRadio.value;
    if (title === null) {
        return;
    }

    fetch("/bookmarks/add-bookmark", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: `title=${title}&url=${url}&tags=${tags}&public=${isPublic}&folder_id=${folderID}`
    }).then(function(response) {
        if (!response.ok) {
            throw new Error(response.statusText);
        }
        return response.json();
    }).then(function(data) {
        const bookmark: Bookmark = {
            ID: data.id,
            Title: data.title,
            URL: data.url,
            Tags: data.tags,
            Public: data.public,
            FolderID: data.folder_id,
            CreatedAt: data.created_at,
        };
        // Add the bookmark to the folder tree and folder-select-tree
        updateNewBookmark(bookmark);
        closeAddBookmarkModal();
    }).catch(function(error) {
        alert(`Failed to add bookmark: ${error}`);
    });
}

// Close the add bookmark modal
function closeAddBookmarkModal() {
    const modalContainer: HTMLDivElement | null = document.getElementById("modal-container") as HTMLDivElement;
    const addBookmarkForm: HTMLFormElement | null = document.getElementById("add-bookmark-form") as HTMLFormElement;

    modalContainer.style.display = "none";
    addBookmarkForm.style.display = "none";
    addBookmarkForm.reset();
}

// Open the edit bookmark modal
function openEditBookmarkModal(event: MouseEvent) {
    const parentFolder: HTMLDivElement | null = (event.target as HTMLButtonElement).closest(".folder");
    const bookmarkLi: HTMLLIElement | null = (event.target as HTMLButtonElement).closest(".bookmark") as HTMLLIElement;
    if (parentFolder === null || bookmarkLi === null) {
        console.error("No parent folder found");
        return;
    }

    const parentFolderID: string | undefined = parentFolder.dataset.id;
    const modalContainer: HTMLDivElement | null = document.getElementById("modal-container") as HTMLDivElement;
    const editBookmarkForm: HTMLFormElement | null = document.getElementById("edit-bookmark-form") as HTMLFormElement;
    const folderSelectWrapper: HTMLDivElement | null = document.getElementById("edit-bookmark-folder-select-wrapper") as HTMLDivElement;
    const folderSelect: HTMLDivElement | null = document.getElementById("folder-select") as HTMLDivElement;
    const bookmarkID: string | undefined = bookmarkLi.dataset.id;
    const bookmarkTitle: HTMLAnchorElement | null = parentFolder.querySelector("a.bookmark-title") as HTMLAnchorElement;
    const bookmarkTags: HTMLSpanElement | null = parentFolder.querySelector("span.bookmark-tags") as HTMLSpanElement;

    // Include the folder select sub-form
    folderSelectWrapper.appendChild(folderSelect);

    // Form elements
    const bookmarkRadio: HTMLInputElement | null =
        editBookmarkForm.elements.namedItem("bookmark_id") as HTMLInputElement;
    const folderRadio: HTMLInputElement | null =
        editBookmarkForm.elements.namedItem("folder-select") as HTMLInputElement;
    const bookmarkTitleInput: HTMLInputElement =
        editBookmarkForm.elements.namedItem("title") as HTMLInputElement;
    const bookmarkUrlInput: HTMLInputElement =
        editBookmarkForm.elements.namedItem("url") as HTMLInputElement;
    const bookmarkTagsInput: HTMLInputElement =
        editBookmarkForm.elements.namedItem("tags") as HTMLInputElement;

    if (parentFolder === null || modalContainer === null || editBookmarkForm === null ||
        folderSelectWrapper === null || bookmarkRadio === null || folderRadio === null ||
        bookmarkTitleInput === null || bookmarkUrlInput === null || bookmarkTagsInput === null) {
        console.error("Missing elements");
        return;
    }

    if (parentFolderID === undefined) {
        console.error("Parent folder ID is undefined");
        return;
    }

    if (bookmarkID === undefined) {
        console.error("Bookmark ID is undefined");
        return;
    }

    editBookmarkForm.reset();
    bookmarkRadio.value = bookmarkID;
    bookmarkTitleInput.value = bookmarkTitle.innerText;
    bookmarkUrlInput.value = bookmarkTitle.href;
    if (bookmarkLi.querySelector("span.bookmark-tags") != null) {
        bookmarkTagsInput.value = bookmarkTags.innerText;
    }
    folderRadio.value = parentFolderID;
    modalContainer.style.display = "block";
    editBookmarkForm.style.display = "block";
    bookmarkTitleInput.focus();
}

// Move a bookmark to a folder when dropped on a folder
function moveBookmarkToFolder(bookmarkID: string, folderID: string) {
    fetch("/bookmarks/move-bookmark", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: `bookmark_id=${bookmarkID}&folder_id=${folderID}`
    }).then(function(response) {
        if (!response.ok) {
            throw new Error(response.statusText);
        }
        return response.json();
    }).then(function(data) {
        if (data.success === true) {
            // Update the folder tree and folder select tree
            const bookmarkLi: HTMLLIElement | null = document.getElementById(`bookmark-${bookmarkID}`) as HTMLLIElement;
            let bookmarkFolder: HTMLLIElement | null, bookmarkUl: HTMLUListElement | null;
            if (folderID === null || folderID === "") {
                bookmarkFolder = document.getElementById(`bookmarks-elements`) as HTMLLIElement;
                bookmarkUl = document.getElementById("bookmarks-root") as HTMLUListElement;
            } else {
                bookmarkFolder = document.getElementById(`folder-${folderID}`) as HTMLLIElement;
                bookmarkUl = bookmarkFolder.querySelector(":scope > details.folder-details > ul.bookmarks") as HTMLUListElement;
            }
            bookmarkUl.appendChild(bookmarkLi);
        } else {
            alert(`Failed to move bookmark: ${data.error}`);
        }
    }).catch(function(error) {
        alert(`Failed to move bookmark: ${error}`);
    });
}

function moveBookmarkFolderToFolder(folderID: string, parentFolderID: string | null) {
    fetch("/bookmarks/move-folder", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: `folder_id=${folderID}&parent_folder_id=${parentFolderID}`
    }).then(function(response) {
        if (!response.ok) {
            throw new Error(response.statusText);
        }
        return response.json();
    }).then(function(data) {
        if (data.success === true) {
            // Update the folder tree and folder select tree
            const folderLi: HTMLLIElement | null = document.getElementById(`folder-${folderID}`) as HTMLLIElement;
            const folderSelectLi: HTMLLIElement | null = document.getElementById(`folder-select-item-${folderID}`) as HTMLLIElement;
            let folderParent: HTMLLIElement | null, folderSelectParent: HTMLLIElement | null;
            if (parentFolderID === null || parentFolderID === "") {
                folderParent = document.getElementById(`folders-list-root`) as HTMLLIElement;
                folderSelectParent = document.getElementById(`folder-select-tree-root`) as HTMLLIElement;
            } else {
                folderParent = document.getElementById(`folders-list-${parentFolderID}`) as HTMLLIElement;
                folderSelectParent = document.getElementById(`folder-select-tree-${parentFolderID}`) as HTMLLIElement;
            }
            folderParent.appendChild(folderLi);
            folderSelectParent.appendChild(folderSelectLi);
        } else {
            alert(`Failed to move folder: ${data.error}`);
        }
    }).catch(function(error) {
        alert(`Failed to move folder: ${error}`);
    });
}

function submitEditBookmarkForm(event: SubmitEvent) {
    const form: HTMLFormElement = event.target as HTMLFormElement;
    const bookmarkIDInput: HTMLInputElement | null = form.elements.namedItem("bookmark_id") as HTMLInputElement;
    const titleInput: HTMLInputElement | null = form.elements.namedItem("title") as HTMLInputElement;
    const urlInput: HTMLInputElement | null = form.elements.namedItem("url") as HTMLInputElement;
    const tagsInput: HTMLInputElement | null = form.elements.namedItem("tags") as HTMLInputElement;
    const isPublicInput: HTMLInputElement | null = form.elements.namedItem("public") as HTMLInputElement;
    const folderRadio: HTMLInputElement | null =
        form.elements.namedItem("folder-select") as HTMLInputElement;

    if (bookmarkIDInput === null || titleInput === null || urlInput === null || tagsInput === null ||
        isPublicInput === null || folderRadio === null) {
        console.error("Missing elements");
        return;
    }

    const bookmarkID: string = bookmarkIDInput.value;
    const title: string = titleInput.value;
    const url: string = urlInput.value;
    const tags: string = tagsInput.value;
    const isPublic: boolean = isPublicInput.checked;
    const folderID: string | null = folderRadio.value === "0" ? null : folderRadio.value;

    if (title === null) {
        return;
    }

    event.preventDefault();

    fetch("/bookmarks/edit-bookmark", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: `bookmark_id=${bookmarkID}&title=${title}&url=${url}&tags=${tags}&public=${isPublic}&folder_id=${folderID}`
    }).then(function(response) {
        if (!response.ok) {
            throw new Error(response.statusText);
        }
        return response.json();
    }).then(function(data) {
        // Update the bookmark li element from the editBookmarkForm
        const bookmarkLi: HTMLLIElement | null = document.getElementById(`bookmark-${data.id}`) as HTMLLIElement;
        const bookmarkTitle: HTMLAnchorElement | null = bookmarkLi.querySelector("a.bookmark-title") as HTMLAnchorElement;
        const bookmarkTags: HTMLSpanElement | null = bookmarkLi.querySelector("span.bookmark-tags") as HTMLSpanElement;

        let parentFolder: HTMLDivElement | null, parentDiv: HTMLDivElement | null;
        if (data.folder_id === null || data.folder_id === undefined || data.folder_id === "") {
            parentFolder = document.getElementById("bookmarks-elements") as HTMLDivElement;
            parentDiv = parentFolder.querySelector(":scope > ul.bookmarks");
        } else {
            parentFolder = document.getElementById(`folder-${data.folder_id}`) as HTMLDivElement;
            parentDiv = parentFolder.querySelector(":scope > details.folder-details > ul.bookmarks");
        }
        bookmarkTitle.innerText = data.title;
        bookmarkTitle.href = data.url;
        bookmarkTags.innerText = data.tags;
        // Update the folder tree and folder select tree
        if (parentDiv !== null) parentDiv.appendChild(bookmarkLi);

        closeEditBookmarkModal();
    }).catch(function(error) {
        alert(`Failed to edit bookmark: ${error}`);
    });
}

// Close the edit bookmark modal
function closeEditBookmarkModal() {
    const modalContainer: HTMLDivElement | null = document.getElementById("modal-container") as HTMLDivElement;
    const editBookmarkForm: HTMLFormElement | null = document.getElementById("edit-bookmark-form") as HTMLFormElement;

    modalContainer.style.display = "none";
    editBookmarkForm.style.display = "none";
    editBookmarkForm.reset();
}

function submitDeleteBookmark(event: Event) {
    const button: HTMLButtonElement | null = event.target as HTMLButtonElement;
    if (button === null) {
        console.error("Missing button");
        return;
    }

    const bookmarkLi: HTMLLIElement | null = button.closest("li.bookmark") as HTMLLIElement;
    if (bookmarkLi === null) {
        console.error("Parent LI is null");
        return;
    }

    const bookmarkID: string | undefined = bookmarkLi.dataset.id;
    if (bookmarkID === undefined) {
        console.error("Bookmark ID is undefined");
        return;
    }

    const bookmarkTitle: HTMLAnchorElement | null = bookmarkLi.querySelector("a.bookmark-title") as HTMLAnchorElement;
    const bookmarkName = bookmarkTitle.innerText;
    if (!confirm(`Are you sure you want to delete the bookmark "${bookmarkName}"?`)) {
        return;
    }

    fetch("/bookmarks/delete-bookmark", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: `bookmark_id=${bookmarkID}`
    }).then(function(response) {
        if (!response.ok) {
            throw new Error(response.statusText);
        }
        return response.json();
    }).then(function(data) {
        if (data.success === true) {
            // Remove the boomark li element from the folder tree
            const bookmarkLi: HTMLLIElement | null = document.getElementById(`bookmark-${bookmarkID}`) as HTMLLIElement;
            if (bookmarkLi !== null) bookmarkLi.remove();
        } else {
            alert(`Failed to delete bookmark: ${data.error}`);
        }
    }).catch(function(error) {
        alert(`Failed to delete bookmark: ${error}`);
    });
}

// Open the add folder modal
function openAddFolderModal(event: Event) {
    const openButton: HTMLButtonElement | null = event.target as HTMLButtonElement;
    const parentFolder: HTMLDivElement | null = openButton.closest(".folder") as HTMLDivElement;
    let parentFolderID: string | undefined = parentFolder.dataset.id;
    const modalContainer: HTMLDivElement | null = document.getElementById("modal-container") as HTMLDivElement;
    const addFolderForm: HTMLFormElement | null = document.getElementById("add-folder-form") as HTMLFormElement;
    const folderSelectWrapper: HTMLDivElement | null = document.getElementById("add-folder-folder-select-wrapper") as HTMLDivElement;
    const folderSelect: HTMLDivElement | null = document.getElementById("folder-select") as HTMLDivElement;

    // Include the folder select sub-form
    folderSelectWrapper.appendChild(folderSelect);

    // Form elements
    const folderRadio: HTMLInputElement | null =
        addFolderForm.elements.namedItem("folder-select") as HTMLInputElement;
    const folderNameInput: HTMLInputElement =
        addFolderForm.elements.namedItem("name") as HTMLInputElement;

    if (folderSelectWrapper === null || addFolderForm === null || folderSelect === null ||
        modalContainer === null || folderRadio === null || folderNameInput === null) {
        console.error("Missing elements");
        return;
    }

    if (parentFolderID === undefined) {
        console.error("Parent folder ID is undefined");
        return;
    }

    addFolderForm.reset();
    folderRadio.value = parentFolderID;
    modalContainer.style.display = "block";
    addFolderForm.style.display = "block";
    folderNameInput.focus();
}

function submitAddFolderForm(event: SubmitEvent) {
    const form: HTMLFormElement = event.target as HTMLFormElement;
    const nameInput: HTMLInputElement | null = form.elements.namedItem("name") as HTMLInputElement;
    const isPublicInput: HTMLInputElement | null = form.elements.namedItem("public") as HTMLInputElement;
    const folderRadio: HTMLInputElement | null =
        form.elements.namedItem("folder-select") as HTMLInputElement;

    const name = nameInput.value;
    const isPublic = isPublicInput.checked;
    const parentFolderID = folderRadio.value === "0" ? null : folderRadio.value;

    if (name === null) {
        return;
    }

    event.preventDefault();

    fetch("/bookmarks/add-folder", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: `name=${name}&public=${isPublic}&parent_folder_id=${parentFolderID}`
    }).then(function(response) {
        if (!response.ok) {
            throw new Error(response.statusText);
        }
        return response.json();
    }).then(function(data) {
        // Add the folder to the folder tree and folder-select-tree
        const folder: BookmarkFolder = {
            ID: data.id,
            Name: data.name,
            ParentFolderID: data.parent_folder_id,
            Public: data.public,
            CreatedAt: data.created_at,
        };
        updateFolder(folder);
        closeAddFolderModal();
    }).catch(function(error) {
        alert(`Failed to add folder: ${error}`);
    });
}

// Close the add folder modal
function closeAddFolderModal() {
    const modalContainer: HTMLDivElement | null = document.getElementById("modal-container") as HTMLDivElement;
    const addFolderForm: HTMLFormElement | null = document.getElementById("add-folder-form") as HTMLFormElement;

    modalContainer.style.display = "none";
    addFolderForm.style.display = "none";
    addFolderForm.reset();
}

function openRenameFolderForm(event: MouseEvent) {
    const button: HTMLButtonElement | null = event.target as HTMLButtonElement;
    const parentFolder: HTMLDivElement | null = button.closest(".folder") as HTMLDivElement;
    const folderID: string | undefined = parentFolder.dataset.id;
    const folderSummary: HTMLElement | null = document.querySelector(`#folder-${folderID} > details.folder-details > summary`) as HTMLElement;
    const folderSpan: HTMLSpanElement | null = document.getElementById(`folder-name-${folderID}`) as HTMLSpanElement;
    const folderActions: HTMLSpanElement | null = document.getElementById(`folder-actions-${folderID}`) as HTMLSpanElement;
    const folderName: string = folderSpan.innerText;
    const renameFolderForm: HTMLFormElement | null = document.createElement("form");

    // Ensure no existing rename folder forms
    document.querySelectorAll("form.rename-folder-form").forEach(form => {
        (form.querySelector("button.cancel-rename-folder") as HTMLButtonElement).click();
    });

    renameFolderForm.setAttribute("id", `rename-folder-form-${folderID}`);
    renameFolderForm.classList.add("rename-folder-form");
    renameFolderForm.innerHTML = `
        <span class="folder-rename-span"><input type="text" name="name" placeholder="Folder name" value="${folderName}" required></span>
        <span class="folder-actions">
            <button type="submit" onsubmit="submitRenameFolderForm(event)">Rename folder</button>
            <button type="button" id="cancel-rename-folder-${folderID}" class="cancel-rename-folder" onclick="closeRenameFolderForm(event)" data-id=${folderID}>Cancel</button>
        </span>
    `;

    const folderNameInput: HTMLInputElement | null = renameFolderForm.elements.namedItem("name") as HTMLInputElement;

    folderSpan.style.display = "none";
    folderActions.style.display = "none";
    folderSummary.appendChild(renameFolderForm);
    renameFolderForm.addEventListener("submit", submitRenameFolderForm);
    folderNameInput.focus();
}

function moveFolderToFolder(folderID: string, parentFolderID: string | null) {
    fetch("/bookmarks/move-folder", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: `folder_id=${folderID}&parent_folder_id=${parentFolderID}`
    }).then(function(response) {
        if (!response.ok) {
            throw new Error(response.statusText);
        }
        return response.json();
    }).then(function(data) {
        if (data.success === true) {
            // Update the folder tree and folder select tree
            const folderLi: HTMLLIElement | null = document.getElementById(`folder-${folderID}`) as HTMLLIElement;
            let folderParent: HTMLLIElement | null;
            if (parentFolderID === null || parentFolderID === "") {
                folderParent = document.getElementById(`folders-list-root`) as HTMLLIElement;
            } else {
                folderParent = document.getElementById(`folders-list-${parentFolderID}`) as HTMLLIElement;
            }
            const folderUl: HTMLUListElement | null = folderParent.querySelector(":scope > ul.folders") as HTMLUListElement;
            folderUl.appendChild(folderLi);
        } else {
            alert(`Failed to move folder: ${data.error}`);
        }
    }).catch(function(error) {
        alert(`Failed to move folder: ${error}`);
    });
}

function submitRenameFolderForm(event: SubmitEvent) {
    const form: HTMLFormElement = event.target as HTMLFormElement;
    const parentFolder: HTMLDivElement | null = form.closest(".folder") as HTMLDivElement;
    const folderID = parentFolder.dataset.id;
    const folderNameSpan: HTMLSpanElement | null = document.getElementById(`folder-name-${folderID}`) as HTMLSpanElement;
    const folderNameInput: HTMLInputElement | null = form.elements.namedItem("name") as HTMLInputElement;
    const folderName: string | undefined = folderNameInput.value;
    if (folderNameInput === null || folderName === undefined) {
        console.error("Missing elements");
        return;
    }
    const folderActions: HTMLSpanElement | null = document.getElementById(`folder-actions-${folderID}`) as HTMLSpanElement;

    event.preventDefault();

    fetch("/bookmarks/rename-folder", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: `folder_id=${folderID}&name=${folderName}`
    }).then(function(response) {
        if (!response.ok) {
            throw new Error(response.statusText);
        }
        return response.json();
    }).then(function(data) {
        // Update the folder name in the folder tree
        const folder: BookmarkFolder = {
            ID: data.id,
            Name: data.name,
            ParentFolderID: data.parent_folder_id,
            Public: data.public,
            CreatedAt: data.created_at,
        };
        updateFolder(folder);

        // Close the form
        //document.getElementById(`cancel-rename-folder-${data.id}`).click();
        form.remove();

        // Show the hidden spans
        folderNameSpan.style.display = "inline";
        folderActions.style.display = "inline";
    }).catch(function(error) {
        alert(`Failed to rename folder: ${error}`);
    });
}

function closeRenameFolderForm(event: MouseEvent) {
    const button: HTMLButtonElement | null = event.target as HTMLButtonElement;
    const folderID: string | undefined = button.dataset.id;
    const form: HTMLFormElement | null = document.getElementById(`rename-folder-form-${folderID}`) as HTMLFormElement;
    const folderName: HTMLSpanElement | null = document.getElementById(`folder-name-${folderID}`) as HTMLSpanElement;
    const folderActions: HTMLSpanElement | null = document.getElementById(`folder-actions-${folderID}`) as HTMLSpanElement;

    form.remove();
    folderName.style.display = "inline";
    folderActions.style.display = "inline";
}

function submitDeleteFolder(event: MouseEvent) {
    const form: HTMLFormElement = event.target as HTMLFormElement;
    const parentFolder: HTMLDivElement | null = form.closest(".folder") as HTMLDivElement;
    if (parentFolder === null) {
        console.error(`Parent folder is null`);
        return;
    }
    const folderID: string | undefined = parentFolder.dataset.id;
    if (folderID === undefined) {
        console.error(`Folder ID is undefined`);
        return;
    }

    const folderDiv: HTMLDivElement | null = document.getElementById(`folder-${folderID}`) as HTMLDivElement;
    if (folderDiv === null) {
        console.error(`Folder does not exist: folder-${folderID}`);
        return;
    }

    const folderName: string = (folderDiv.querySelector(`:scope > details.folder-details > summary > span.folder-name`) as HTMLSpanElement).innerText;
    if (!confirm(`Are you sure you want to delete the folder "${folderName}"?`)) {
        return;
    }

    fetch("/bookmarks/delete-folder", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: `folder_id=${folderID}`
    }).then(function(response) {
        if (!response.ok) {
            throw new Error(response.statusText);
        }
        return response.json();
    }).then(function(data) {
        const folderLi: HTMLLIElement | null = document.getElementById(`folder-${folderID}`) as HTMLLIElement;
        const folderSelectLi: HTMLLIElement | null = document.getElementById(`folder-select-item-${folderID}`) as HTMLLIElement;
        // Remove the folder details element from the folder tree
        folderLi.remove();
        // Remove the folder from the folder select tree
        folderSelectLi.remove();
    }).catch(function(error) {
        alert(`Failed to delete folder: ${error}`);
    });
}

// Update the folder tree after adding a new bookmark
function updateNewBookmark(bookmark: Bookmark) {
    updateFolderTreeWithBookmark(bookmark);
}

function updateFolderTreeWithBookmark(bookmark: Bookmark) {
    let parentFolder: HTMLDivElement | null;
    if (bookmark.FolderID === null || bookmark.FolderID === undefined || bookmark.FolderID === "") {
        parentFolder = document.getElementById("bookmarks-root") as HTMLDivElement;
    } else {
        parentFolder = document.getElementById(`bookmarks-${bookmark.FolderID}`) as HTMLDivElement;
    }

    if (parentFolder === null) {
        console.error(`Folder does not exist: bookmarks-${bookmark.FolderID}`);
        return;
    }

    let bookmarkLi = document.getElementById(`bookmark-${bookmark.ID}`) as HTMLLIElement;
    if (bookmarkLi !== null) {
        const bookmarkTitle: HTMLAnchorElement | null = bookmarkLi.querySelector("a.bookmark-title") as HTMLAnchorElement;
        const bookmarkTags: HTMLSpanElement | null = bookmarkLi.querySelector("span.bookmark-tags") as HTMLSpanElement;
        // Set name
        bookmarkTitle.innerText = bookmark.Title;

        if (typeof bookmark.URL !== "undefined" && bookmark.URL !== null) {
            bookmarkTitle.href = bookmark.URL;
        } else {
            bookmarkTitle.href = "";
        }

        if (typeof bookmark.Tags !== "undefined" && bookmark.Tags !== null) {
            bookmarkTags.innerText = bookmark.Tags;
        } else {
            bookmarkTags.innerText = "";
        }
    } else {
        bookmarkLi = document.createElement("li");
        bookmarkLi.dataset.id = bookmark.ID.toString();
        bookmarkLi.addEventListener("dragstart", dragStartBookmark);
        bookmarkLi.setAttribute("id", `bookmark-${bookmark.ID}`);
        bookmarkLi.classList.add("bookmark");
        bookmarkLi.dataset.id = bookmark.ID.toString();
        bookmarkLi.setAttribute("draggable", "true");

        const bookmarkTitleSpan: HTMLSpanElement | null = document.createElement("span");
        bookmarkTitleSpan.classList.add("bookmark-title-span");
        bookmarkTitleSpan.setAttribute("draggable", "false");
        bookmarkLi.appendChild(bookmarkTitleSpan);

        const bookmarkTitle: HTMLAnchorElement | null = document.createElement("a");
        bookmarkTitle.href = bookmark.URL;
        bookmarkTitle.target = "_blank";
        bookmarkTitle.classList.add("bookmark-title");
        bookmarkTitle.innerText = bookmark.Title;
        bookmarkTitleSpan.appendChild(bookmarkTitle);

        const bookmarkTags: HTMLSpanElement | null = document.createElement("span");
        if (typeof bookmark.Tags !== "undefined" && bookmark.Tags !== null) {
            bookmarkTags.classList.add("bookmark-tags");
            bookmarkTags.innerText = bookmark.Tags;
        }
        bookmarkLi.appendChild(bookmarkTags);

        const bookmarkActions: HTMLSpanElement | null = document.createElement("span");
        bookmarkActions.classList.add("bookmark-actions");
        const editBookmarkButton: HTMLButtonElement | null = document.createElement("button");
        editBookmarkButton.classList.add("edit-bookmark");
        editBookmarkButton.innerText = "Edit";
        editBookmarkButton.addEventListener("click", openEditBookmarkModal);
        const deleteBookmarkButton: HTMLButtonElement | null = document.createElement("button");
        deleteBookmarkButton.classList.add("delete-bookmark");
        deleteBookmarkButton.innerText = "Delete";
        deleteBookmarkButton.addEventListener("click", submitDeleteBookmark);
        bookmarkActions.appendChild(editBookmarkButton);
        bookmarkActions.appendChild(deleteBookmarkButton);
        bookmarkLi.appendChild(bookmarkActions);

        parentFolder.appendChild(bookmarkLi);
    }

    try {
        parentFolder.appendChild(bookmarkLi);
    } catch (error) {
        console.error(`Failed to add bookmark ${bookmark.Title} to folder tree ${bookmark.FolderID}: ${error}`);
    }
}

// Update the folder tree and folder select tree after adding a new folder
function updateFolder(folder: BookmarkFolder) {
    updateFolderTreeWithFolder(folder);
    updateFolderSelectTreeWithFolder(folder);
}

// Add a folder from Object
function updateFolderTreeWithFolder(folder: BookmarkFolder) {
    let parentFolder: HTMLDivElement | null;
    if (folder.ParentFolderID === null || folder.ParentFolderID === undefined || folder.ParentFolderID === "") {
        parentFolder = document.getElementById(`folders-list-root`) as HTMLDivElement;
    } else {
        parentFolder = document.getElementById(`folders-list-${folder.ParentFolderID}`) as HTMLDivElement;
    }

    if (parentFolder === null) {
        console.error(`Folder does not exist: folders-list-${folder.ParentFolderID}`);
        return;
    }

    let folderLi: HTMLLIElement | null = document.getElementById(`folder-${folder.ID}`) as HTMLLIElement;
    if (folderLi !== null) {
        const folderName: HTMLSpanElement | null = folderLi.querySelector("span.folder-name") as HTMLSpanElement;
        if (folderName === null) {
            console.error(`Folder name is null`);
            return;
        }
        // Set name
        folderName.innerText = folder.Name;
    } else {
        folderLi = document.createElement("li");
        folderLi.dataset.id = folder.ID.toString();
        folderLi.setAttribute("draggable", "true");
        folderLi.addEventListener("dragstart", dragStartFolder);
        folderLi.setAttribute("id", `folder-${folder.ID}`);
        folderLi.classList.add("folder");
        folderLi.dataset.id = folder.ID;

        const folderDetails: HTMLDetailsElement | null = document.createElement("details");
        const folderSummary: HTMLElement | null = document.createElement("summary");
        const openFolderIcon: HTMLSpanElement | null = document.createElement("span");
        const closedFolderIcon: HTMLSpanElement | null = document.createElement("span");
        const folderName: HTMLSpanElement | null = document.createElement("span");
        const folderActions: HTMLSpanElement | null = document.createElement("span");
        const addBookmarkButton: HTMLButtonElement | null = document.createElement("button");
        const addFolderButton: HTMLButtonElement | null = document.createElement("button");
        const renameFolderButton: HTMLButtonElement | null = document.createElement("button");
        const deleteFolderButton: HTMLButtonElement | null = document.createElement("button");
        const folderUl: HTMLUListElement | null = document.createElement("ul");
        const bookmarksUl: HTMLUListElement | null = document.createElement("ul");

        // Set folder icons
        closedFolderIcon.classList.add("folder-icon-closed");
        closedFolderIcon.innerText = "📁";
        openFolderIcon.classList.add("folder-icon-open");
        openFolderIcon.innerText = "📂";

        // Set folder name
        folderName.innerText = folder.Name;
        folderName.setAttribute("id", `folder-name-${folder.ID}`);
        folderName.classList.add("folder-name");

        // Set folder data-id
        folderDetails.classList.add("folder-details");
        folderDetails.dataset.id = folder.ID;

        // Set folder actions
        folderActions.setAttribute("id", `folder-actions-${folder.ID}`);
        folderActions.classList.add("folder-actions");

        addBookmarkButton.classList.add("add-bookmark");
        addBookmarkButton.innerText = "+ Bookmark";
        addBookmarkButton.addEventListener("click", openAddBookmarkModal);
        folderActions.appendChild(addBookmarkButton);

        addFolderButton.classList.add("add-folder");
        addFolderButton.innerText = "+ Folder";
        addFolderButton.addEventListener("click", openAddFolderModal);
        folderActions.appendChild(addFolderButton);

        renameFolderButton.classList.add("rename-folder");
        renameFolderButton.innerText = "Rename";
        renameFolderButton.addEventListener("click", openRenameFolderForm);
        folderActions.appendChild(renameFolderButton);

        deleteFolderButton.classList.add("delete-folder");
        deleteFolderButton.innerText = "Delete";
        deleteFolderButton.addEventListener("click", submitDeleteFolder);
        folderActions.appendChild(deleteFolderButton);

        // Set folder ul class
        folderUl.setAttribute("id", `folders-list-${folder.ID}`);
        folderUl.classList.add("folders");

        // Set bookmarks ul class
        bookmarksUl.setAttribute("id", `bookmarks-${folder.ID}`);
        bookmarksUl.classList.add("bookmarks");

        // Set folder summary
        folderSummary.appendChild(closedFolderIcon);
        folderSummary.appendChild(openFolderIcon);
        folderSummary.appendChild(folderName);
        folderSummary.appendChild(folderActions);
        // Set folder details
        folderDetails.appendChild(folderSummary);
        folderDetails.appendChild(folderUl);
        folderDetails.appendChild(bookmarksUl);

        // Set folder li
        folderLi.appendChild(folderDetails);
    }

    try {
        parentFolder.appendChild(folderLi);
    } catch (error) {
        console.error(`Failed to add folder to folder tree: ${error}`);
    }
}

function updateFolderSelectTreeWithFolder(folder: BookmarkFolder) {
    let parentDiv: HTMLDivElement | null;
    if (folder.ParentFolderID === null || folder.ParentFolderID === undefined || folder.ParentFolderID === "") {
        parentDiv = document.getElementById("folder-select-tree-root") as HTMLDivElement
    } else {
        parentDiv = document.getElementById(`folder-select-tree-${folder.ParentFolderID}`) as HTMLDivElement;
    }
    let folderSelectLi: HTMLLIElement | null = document.getElementById(`folder-select-item-${folder.ID}`) as HTMLLIElement;

    if (parentDiv === null) {
        console.error(`Folder does not exist: folder-select-tree-${folder.ParentFolderID}`);
        return;
    }

    if (folderSelectLi !== null) {
        const folderSelectLabel: HTMLLabelElement | null = folderSelectLi.querySelector("label") as HTMLLabelElement;
        // Set name
        folderSelectLabel.innerText = folder.Name;
    } else {
        folderSelectLi = document.createElement("li");
        folderSelectLi.setAttribute("id", `folder-select-item-${folder.ID}`);
        folderSelectLi.classList.add("folder-select");
        folderSelectLi.innerHTML = `
            <input type="radio" name="folder-select" id="folder-select-${folder.ID}" value="${folder.ID}">
            <label for="folder-select-${folder.ID}">📁 ${folder.Name}</label>
        `;

        const folderSelectTree: HTMLUListElement | null = document.createElement("ul");
        folderSelectTree.setAttribute("id", `folder-select-tree-${folder.ID}`);
        folderSelectTree.classList.add("folder-select-tree");
        folderSelectTree.appendChild(folderSelectLi);

        parentDiv.appendChild(folderSelectTree);
    }

    try {
        parentDiv.appendChild(folderSelectLi);
    } catch (error) {
        console.error(`Failed to add folder ${folder.Name} to folder select tree ${folder.ParentFolderID}: ${error}`);
    }
}

