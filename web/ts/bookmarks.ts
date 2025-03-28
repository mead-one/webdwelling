export {};

document.addEventListener("DOMContentLoaded", function() {
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

    if (folderSelectWrapper === null || addBookmarkForm === null || folderSelect === null || modalContainer === null) {
        console.error("Missing elements");
        return;
    }

    folderSelectWrapper.appendChild(folderSelect);
    addBookmarkForm.reset();
    addBookmarkForm.elements["folder-select"].value = parentFolderID ? parentFolderID : "";
    modalContainer.style.display = "block";
    addBookmarkForm.style.display = "block";
    addBookmarkForm.elements["title"].focus();
}

function submitAddBookmarkForm(event: SubmitEvent) {
    event.preventDefault();
    const form: HTMLFormElement = event.target as HTMLFormElement;
    const title = form.elements["title"].value;
    const url = form.elements["url"].value;
    const tags = form.elements["tags"].value;
    const isPublic = form.elements["public"].checked;
    const folderID = form.elements["folder-select"].value === "0" ? null : form.elements["folder-select"].value;
    if (title === null) {
        return;
    }

    console.log(`Adding bookmark ${title} to folder ${folderID}`);

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
    const parentFolder: HTMLDivElement | null = (event.target as HTMLFormElement).closest(".folder");
    if (parentFolder === null) {
        console.error("No parent folder found");
        return;
    }

    const parentFolderID: string | undefined = parentFolder.dataset.id;
    const modalContainer: HTMLDivElement | null = document.getElementById("modal-container") as HTMLDivElement;
    const editBookmarkForm: HTMLFormElement | null = document.getElementById("edit-bookmark-form") as HTMLFormElement;
    const folderSelectWrapper: HTMLDivElement | null = document.getElementById("edit-bookmark-folder-select-wrapper") as HTMLDivElement;
    const folderSelect: HTMLDivElement | null = document.getElementById("folder-select") as HTMLDivElement;
    const bookmarkLi: HTMLLIElement | null = parentFolder.closest(".bookmark") as HTMLLIElement;
    const bookmarkTitle: HTMLAnchorElement | null = parentFolder.querySelector("a.bookmark-title") as HTMLAnchorElement;
    const bookmarkTags: HTMLSpanElement | null = parentFolder.querySelector("span.bookmark-tags") as HTMLSpanElement;

    folderSelectWrapper.appendChild(folderSelect);
    editBookmarkForm.reset();
    editBookmarkForm.elements["bookmark_id"].value = bookmarkLi.dataset.id;
    editBookmarkForm.elements["title"].value = bookmarkTitle.innerText;
    editBookmarkForm.elements["url"].value = bookmarkTitle.href;
    if (bookmarkLi.querySelector("span.bookmark-tags") != null) {
        editBookmarkForm.elements["tags"].value = bookmarkTags.innerText;
    }
    editBookmarkForm.elements["folder-select"].value = parentFolderID;
    modalContainer.style.display = "block";
    editBookmarkForm.style.display = "block";
    editBookmarkForm.elements["title"].focus();
}

function submitEditBookmarkForm(event: SubmitEvent) {
    const form: HTMLFormElement = event.target as HTMLFormElement;
    const title = form.elements["title"].value;
    const url = form.elements["url"].value;
    const tags = form.elements["tags"].value;
    const isPublic = form.elements["public"].checked;
    const folderID = form.elements["folder-select"].value === "0" ? null : form.elements["folder-select"].value;
    if (title === null) {
        return;
    }

    event.preventDefault();

    console.log(`Editing bookmark ${title} to folder ${folderID}`);

    fetch("/bookmarks/edit-bookmark", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: `bookmark_id=${form.elements["bookmark_id"].value}&title=${title}&url=${url}&tags=${tags}&public=${isPublic}&folder_id=${folderID}`
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
    const form: HTMLFormElement = event.target as HTMLFormElement;
    const bookmarkID: string = form.elements["bookmark_id"].value;
    if (bookmarkID === null) {
        return;
    }

    const bookmarkLi: HTMLLIElement | null = document.getElementById(`bookmark-${bookmarkID}`) as HTMLLIElement;
    const bookmarkTitle: HTMLAnchorElement | null = bookmarkLi.querySelector("a.bookmark-title") as HTMLAnchorElement;
    const bookmarkName = bookmarkTitle.innerText;
    if (!confirm(`Are you sure you want to delete the bookmark "${bookmarkName}"?`)) {
        return;
    }

    console.log(`Deleting bookmark ${bookmarkID}`);

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

    folderSelectWrapper.appendChild(folderSelect);
    addFolderForm.reset();
    addFolderForm.elements["folder-select"].value = parentFolderID;
    modalContainer.style.display = "block";
    addFolderForm.style.display = "block";
    addFolderForm.elements["name"].focus();
}

function submitAddFolderForm(event: SubmitEvent) {
    const form: HTMLFormElement = event.target as HTMLFormElement;
    const name = form.elements["name"].value;
    const isPublic = form.elements["public"].checked;
    const parentFolderID = form.elements["folder-select"].value === "0" ? null : form.elements["folder-select"].value;
    if (name === null) {
        return;
    }

    console.log(`Adding folder ${name} to folder ${parentFolderID}`);
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

    folderSpan.style.display = "none";
    folderActions.style.display = "none";
    folderSummary.appendChild(renameFolderForm);
    renameFolderForm.addEventListener("submit", submitRenameFolderForm);
    renameFolderForm.elements["name"].focus();
}

function submitRenameFolderForm(event: SubmitEvent) {
    const form: HTMLFormElement = event.target as HTMLFormElement;
    const parentFolder: HTMLDivElement | null = form.closest(".folder") as HTMLDivElement;
    const folderID = parentFolder.dataset.id;
    const folderName = form.elements["name"].value;
    if (folderName === null) {
        return;
    }
    const folderActions: HTMLSpanElement | null = document.getElementById(`folder-actions-${folderID}`) as HTMLSpanElement;

    console.log(`Renaming folder ${folderID} to ${folderName}`);
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
        folderName.style.display = "inline";
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

    console.log(`Deleting folder ${folderID}`);

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
        bookmarkLi.setAttribute("id", `bookmark-${bookmark.ID}`);
        bookmarkLi.classList.add("bookmark");
        bookmarkLi.dataset.id = bookmark.ID.toString();

        const bookmarkTitle: HTMLAnchorElement | null = document.createElement("a");
        bookmarkTitle.href = bookmark.URL;
        bookmarkTitle.target = "_blank";
        bookmarkTitle.classList.add("bookmark-title");
        bookmarkTitle.innerText = bookmark.Title;
        bookmarkLi.appendChild(bookmarkTitle);

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

