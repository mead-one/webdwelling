document.addEventListener("DOMContentLoaded", function() {
    const bookmarkButtons = document.querySelectorAll("button.add-bookmark");
    const deleteBookmarkButtons = document.querySelectorAll("button.delete-bookmark");
    const folderButtons = document.querySelectorAll("button.add-folder");
    const renameFolderButtons = document.querySelectorAll("button.rename-folder");
    const deleteFolderButtons = document.querySelectorAll("button.delete-folder");
    const addBookmarkForm = document.getElementById("add-bookmark-form");
    const addFolderForm = document.getElementById("add-folder-form");
    const cancelAddBookmarkButton = document.getElementById("cancel-add-bookmark");
    const cancelAddFolderButton = document.getElementById("cancel-add-folder");

    // Add bookmark buttons
    bookmarkButtons.forEach(button => {
        button.addEventListener("click", openAddBookmarkModal);
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

    addBookmarkForm.addEventListener("submit", submitAddBookmarkForm);
    addFolderForm.addEventListener("submit", submitAddFolderForm);
    cancelAddBookmarkButton.addEventListener("click", closeAddBookmarkModal);
    cancelAddFolderButton.addEventListener("click", closeAddFolderModal);
});

// Open the add bookmark modal
function openAddBookmarkModal(event) {
    let parentFolderID;
    if (event.target.parentElement.parentElement.id === "bookmarks-elements") {
        parentFolderID = 0;
    } else {
        parentFolderID = event.target.parentElement.dataset.id;
    }

    const modalContainer = document.getElementById("modal-container");
    const addBookmarkForm = document.getElementById("add-bookmark-form");
    const folderSelectWrapper = document.getElementById("add-bookmark-folder-select-wrapper");
    const folderSelect = document.getElementById("folder-select");

    folderSelectWrapper.appendChild(folderSelect);
    addBookmarkForm.reset();
    addBookmarkForm.elements["folder-select"].value = parentFolderID;
    modalContainer.style.display = "block";
    addBookmarkForm.style.display = "block";
    addBookmarkForm.elements["title"].focus();
}

function submitAddBookmarkForm(event) {
    event.preventDefault();
    const title = event.target.elements["title"].value;
    const url = event.target.elements["url"].value;
    const tags = event.target.elements["tags"].value;
    const public = event.target.elements["public"].checked;
    const folderID = event.target.elements["folder-select"].value === "0" ? null : event.target.elements["folder-select"].value;
    if (title === null) {
        return;
    }

    console.log(`Adding bookmark ${title} to folder ${folderID}`);

    fetch("/bookmarks/add-bookmark", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: `title=${title}&url=${url}&tags=${tags}&public=${public}&folder_id=${folderID}`
    }).then(function(response) {
        if (!response.ok) {
            throw new Error(response.statusText);
        }
        return response.json();
    }).then(function(data) {
        // Check the response status
        console.log(data);

        // Add the bookmark to the folder tree and folder-select-tree
        updateNewBookmark(data.title, data.url, data.tags, data.folder_id, data.id);
        closeAddBookmarkModal();
    }).catch(function(error) {
        alert(`Failed to add bookmark: ${error}`);
    });
}

// Close the add bookmark modal
function closeAddBookmarkModal() {
    const modalContainer = document.getElementById("modal-container");
    const addBookmarkForm = document.getElementById("add-bookmark-form");

    modalContainer.style.display = "none";
    addBookmarkForm.style.display = "none";
    addBookmarkForm.reset();
}

function submitDeleteBookmark(event) {
    const bookmarkID = event.target.closest("li.bookmark").dataset.id;
    if (bookmarkID === null) {
        return;
    }

    const bookmarkName = document.querySelector(`#bookmark-${bookmarkID} .bookmark-title`).innerText;
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
        // Check the response status
        console.log(data);

        // Remove the boomark li element from the folder tree
        document.getElementById(`bookmark-${bookmarkID}`).remove();
    }).catch(function(error) {
        alert(`Failed to delete bookmark: ${error}`);
    });
}

// Open the add folder modal
function openAddFolderModal(event) {
    let parentFolderID;
    if (event.target.parentElement.parentElement.id === "bookmarks-elements") {
        parentFolderID = 0;
    } else {
        parentFolderID = event.target.closest("details.folder").dataset.id;
    }

    const modalContainer = document.getElementById("modal-container");
    const addFolderForm = document.getElementById("add-folder-form");
    const folderSelectWrapper = document.getElementById("add-folder-folder-select-wrapper");
    const folderSelect = document.getElementById("folder-select");

    folderSelectWrapper.appendChild(folderSelect);
    addFolderForm.reset();
    addFolderForm.elements["folder-select"].value = parentFolderID;
    modalContainer.style.display = "block";
    addFolderForm.style.display = "block";
    addFolderForm.elements["name"].focus();
}

function submitAddFolderForm(event) {
    event.preventDefault();
    const name = event.target.elements["name"].value;
    const public = event.target.elements["public"].checked;
    const parentFolderID = event.target.elements["folder-select"].value === "0" ? null : event.target.elements["folder-select"].value;
    if (name === null) {
        return;
    }

    console.log(`Adding folder ${name} to folder ${parentFolderID}`);

    fetch("/bookmarks/add-folder", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: `name=${name}&public=${public}&parent_folder_id=${parentFolderID}`
    }).then(function(response) {
        if (!response.ok) {
            throw new Error(response.statusText);
        }
        return response.json();
    }).then(function(data) {
        // Check the response status
        console.log(data);

        // Add the folder to the folder tree and folder-select-tree
        updateNewFolder(data.name, data.parent_folder_id, data.id);
        closeAddFolderModal();
    }).catch(function(error) {
        alert(`Failed to add folder: ${error}`);
    });
}

// Close the add folder modal
function closeAddFolderModal() {
    const modalContainer = document.getElementById("modal-container");
    const addFolderForm = document.getElementById("add-folder-form");

    modalContainer.style.display = "none";
    addFolderForm.style.display = "none";
    addFolderForm.reset();
}

function openRenameFolderForm(event) {
    console.log(`Renaming folder ${event.target.closest("details.folder").dataset.id}`);
    const folderID = event.target.closest("details.folder").dataset.id;
    const folderSummary = document.querySelector(`#folder-${folderID} > summary`);
    const folderSpan = folderSummary.querySelector("span.folder-name");
    const folderActions = folderSummary.querySelector("span.folder-actions");
    const folderName = folderSpan.innerText.slice(3);
    const renameFolderForm = document.createElement("form");
    renameFolderForm.id = `rename-folder-form-${folderID}`;
    renameFolderForm.classList.add("rename-folder-form");
    renameFolderForm.innerHTML = `
        <span class="folder-rename-span"><input type="text" name="name" placeholder="Folder name" value="${folderName}" required></span>
        <span class="folder-actions">
            <button type="submit">Rename folder</button>
            <button type="button" id="cancel-rename-folder-${folderID}" class="cancel-rename-folder">Cancel</button>
        </span>
    `;

    folderSpan.style.display = "none";
    folderActions.style.display = "none";
    folderSummary.appendChild(renameFolderForm);
    renameFolderForm.addEventListener("submit", submitRenameFolderForm);
    renameFolderForm.querySelector("button.cancel-rename-folder").addEventListener("click", closeRenameFolderForm);
}

function closeRenameFolderForm(event) {
    const folderID = event.target.closest("details.folder").dataset.id;
    const folderSummary = document.querySelector(`#folder-${folderID} > summary`);
    const renameFolderForm = document.getElementById(`rename-folder-form-${folderID}`);
    const folderSpan = folderSummary.querySelector("span.folder-name");
    const folderActions = folderSummary.querySelector("span.folder-actions");

    folderSpan.style.display = "inline";
    folderActions.style.display = "inline";
    renameFolderForm.remove();
}

function submitRenameFolderForm(event) {
    event.preventDefault();
    const folderID = event.target.closest("details.folder").dataset.id;
    const folderName = event.target.elements["name"].value;
    if (folderName === null) {
        return;
    }

    console.log(`Renaming folder ${folderID} to ${folderName}`);

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
        // Check the response status
        console.log(data);

        // Update the folder name in the folder tree
        document.getElementById(`folder-${folderID}`).querySelector("span.folder-name").innerText = `📁 ${folderName}`;

        // Update the folder name in the folder select tree
        const folderSelectTree = document.getElementById(`folder-select-tree-${folderID}`);
        const folderSelectLi = folderSelectTree.querySelector("li.folder-select");
        folderSelectLi.querySelector("label").innerText = `📁 ${folderName}`;

        // Update the folder name in the modal
        const modalFolderName = document.getElementById(`folder-name-${folderID}`);
        modalFolderName.innerText = `📁 ${folderName}`;
    }).catch(function(error) {
        alert(`Failed to rename folder: ${error}`);
    });
}

function submitDeleteFolder(event) {
    const folderID = event.target.closest("details.folder").dataset.id;
    if (folderID === null) {
        return;
    }

    const folderName = document.querySelector(`#folder-${folderID} > summary > span.folder-name`).innerText;
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
        // Check the response status
        console.log(data);

        // Remove the folder details element from the folder tree
        document.getElementById(`folder-${folderID}`).remove();
    }).catch(function(error) {
        alert(`Failed to delete folder: ${error}`);
    });
}

// Add a folder from Object
function addFolderFromObject(folder) {
    // Create folder elements
    const folderTree = document.createElement("ul");
    const folderLi = document.createElement("li");
    const folderDetails = document.createElement("details");
    const folderSummary = document.createElement("summary");
    const folderName = document.createElement("span");
    const folderActions = document.createElement("span");

    // Set foldertree ID and class
    folderTree.id = `folder-${folder.id}`;
    folderTree.classList.add("foldertree");

    // set folder li class
    folderLi.classList.add("folder");

    // Set folder name
    folderName.innerText = folder.name;
    folderName.classList.add("folder-name");

    // Set folder data-id
    folderDetails.dataset.id = folder.id;
    // Set folder actions
    folderActions.innerHTML = `
        <button class="add-bookmark">+ Bookmark</button>
        <button class="add-folder">+ Folder</button>
    `;

    // Set folder summary
    folderSummary.appendChild(folderName);
    folderSummary.appendChild(folderActions);

    // Set folder details
    folderDetails.appendChild(folderSummary);
    folderDetails.appendChild(document.createElement("ul"));
    folderLi.appendChild(folderDetails);
    folderTree.appendChild(folderLi);

    // Add folder to folder tree
    const parentDiv = document.getElementById(`folder-${folder.parent_folder_id}`);
    parentDiv.appendChild(folderDetails);
}

// Update the folder tree after adding a new bookmark
function updateNewBookmark(title, url, tags, folderID, newID) {
    addBookmarkToFolderTree(title, url, tags, folderID, newID);
}

// Add a bookmark to the bookmark tree
function addBookmarkToFolderTree(title, url, tags, folderID, newID) {
    let parentFolder;
    if (folderID === null || folderID === undefined || folderID === "") {
        parentFolder = document.getElementById("bookmarks-elements");
    } else {
        parentFolder = document.getElementById(`folder-${folderID}`);
    }
    const parentDiv = parentFolder.querySelector(":scope > ul.bookmarks");
    const bookmarkLi = document.createElement("li");

    bookmarkLi.id = `bookmark-${newID}`;
    bookmarkLi.classList.add("bookmark");

    const bookmarkLink = document.createElement("a");
    bookmarkLink.href = url;
    bookmarkLink.target = "_blank";
    bookmarkLink.classList.add("bookmark-title");
    bookmarkLink.innerText = title;
    bookmarkLi.appendChild(bookmarkLink);

    if (tags !== null && tags !== undefined) {
        bookmarkTags = document.createElement("span");
        bookmarkTags.classList.add("bookmark-tags");
        bookmarkTags.innerText = tags;
        bookmarkLi.appendChild(bookmarkTags);
    }

    const bookmarkActions = document.createElement("span");
    bookmarkActions.classList.add("bookmark-actions");
    const editBookmarkButton = document.createElement("button");
    editBookmarkButton.classList.add("edit-bookmark");
    editBookmarkButton.innerText = "Edit";
    //editBookmarkButton.addEventListener("click", openEditBookmarkModal);
    const deleteBookmarkButton = document.createElement("button");
    deleteBookmarkButton.classList.add("delete-bookmark");
    deleteBookmarkButton.innerText = "Delete";
    //deleteBookmarkButton.addEventListener("click", openDeleteBookmarkModal);
    bookmarkActions.appendChild(editBookmarkButton);
    bookmarkActions.appendChild(deleteBookmarkButton);
    bookmarkLi.appendChild(bookmarkActions);

    parentDiv.appendChild(bookmarkLi);
}

// Update the folder tree and folder select tree after adding a new folder
function updateNewFolder(name, parentFolderID, newID) {
    console.log(`Params: name: ${name}, parentFolderID: ${parentFolderID}, newID: ${newID}`);
    addFolderToFolderTree(name, parentFolderID, newID);
    addFolderToFolderSelectTree(name, parentFolderID, newID);
}

// Add a folder to the folder tree
function addFolderToFolderTree(name, parentFolderID, newID) {
    let parentFolder;
    if (parentFolderID === null || parentFolderID === undefined || parentFolderID === "") {
        parentFolder = document.getElementById("bookmarks-elements");
    } else {
        parentFolder = document.getElementById(`folder-${parentFolderID}`);
    }
    const bookmarksUl = parentFolder.querySelector(":scope > ul.bookmarks");
    const folderDetails = document.createElement("details");
    const folderSummary = document.createElement("summary");
    const folderName = document.createElement("span");
    const folderActions = document.createElement("span");
    const addFolderButton = document.createElement("button");
    const addBookmarkButton = document.createElement("button");
    const renameFolderButton = document.createElement("button");
    const deleteFolderButton = document.createElement("button");
    const folderUl = document.createElement("ul");

    // Set folder name
    folderName.innerText = `📁 ${name}`;
    folderName.classList.add("folder-name");

    // Set folder data-id
    folderDetails.id = `folder-${newID}`;
    folderDetails.classList.add("folder");
    folderDetails.dataset.id = newID;

    // Set folder actions
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
    folderUl.classList.add("bookmarks");

    // Set folder summary
    folderSummary.appendChild(folderName);
    folderSummary.appendChild(folderActions);
    // Set folder details
    folderDetails.appendChild(folderSummary);
    folderDetails.appendChild(folderUl);

    // Add folder to folder tree
    parentFolder.appendChild(folderDetails);
    // Place the bookmarks ul at the end of the folder tree
    bookmarksUl.parentElement.appendChild(bookmarksUl);
}

function addFolderToFolderSelectTree(name, parentFolderID, newID) {
    console.log(`Params: name: ${name}, parentFolderID: ${parentFolderID}, newID: ${newID}`);
    let parentDiv;
    if (parentFolderID === null || parentFolderID === undefined || parentFolderID === "") {
        parentDiv= document.getElementById("folder-select").querySelector(":scope > ul.folder-select-tree");
    } else {
        parentDiv = document.getElementById(`folder-select-tree-${parentFolderID}`)
    }
    const folderSelect = document.createElement("input");

    folderSelect.type = "radio";
    folderSelect.name = "folder-select";
    folderSelect.value = newID;
    folderSelect.id = `folder-select-${newID}`;

    const folderSelectLabel = document.createElement("label");
    folderSelectLabel.innerText = `📁 ${name}`;
    folderSelectLabel.htmlFor = folderSelect.id;

    console.log(`Folder select ID: ${folderSelect.id}, folder select label for: ${folderSelectLabel.htmlFor}`);

    const folderSelectLi = document.createElement("li");
    folderSelectLi.classList.add("folder-select");
    folderSelectLi.appendChild(folderSelect);
    folderSelectLi.appendChild(folderSelectLabel);

    const folderSelectTree = document.createElement("ul");
    folderSelectTree.id = `folder-select-tree-${newID}`;
    folderSelectTree.classList.add("folder-select-tree");
    folderSelectTree.appendChild(folderSelectLi);

    parentDiv.appendChild(folderSelectTree);
}
