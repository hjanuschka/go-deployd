// Process data when retrieving pets
// Available variables: this (data), me (user session), query, isRoot

// Hide sensitive information from non-authenticated users
if (!me && !isRoot) {
    hide('createdBy');
    hide('updatedBy');
    hide('notes'); // Private notes only for authenticated users
}

// Add computed fields
if (this.age) {
    if (this.age < 1) {
        this.ageCategory = 'puppy/kitten';
    } else if (this.age < 3) {
        this.ageCategory = 'young';
    } else if (this.age < 8) {
        this.ageCategory = 'adult';
    } else {
        this.ageCategory = 'senior';
    }
}

// Add availability status
this.isAvailableForAdoption = !this.owner || this.owner.trim() === '';

// Format dates for display
if (this.adoptionDate) {
    this.adoptionDateFormatted = new Date(this.adoptionDate).toLocaleDateString();
}

if (this.createdAt) {
    this.createdAtFormatted = new Date(this.createdAt).toLocaleDateString();
}

// Access logged for audit trail
deployd.log("Pet data accessed", {
    petId: data.id,
    user: me,
    hiddenFields: !me
});