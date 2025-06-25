// Process data when updating pets
// Available variables: this (data), me (user session), query, isRoot

// Track update metadata
this.updatedBy = me ? me.id : 'anonymous';
this.updatedAt = new Date().toISOString();

// Validate user permissions - only authenticated users can update
if (!me && !isRoot) {
    cancel('Authentication required to update pets');
}

// Business logic - update adoption status based on owner
if (this.owner && this.owner.trim() !== '') {
    this.adoptionStatus = 'adopted';
    if (!this.adoptionDate) {
        this.adoptionDate = new Date().toISOString();
    }
} else {
    this.adoptionStatus = 'available';
    this.adoptionDate = null;
}

// Prevent certain fields from being modified after creation
if (this.petId && query.originalData && query.originalData.petId && this.petId !== query.originalData.petId) {
    error('petId', 'Pet ID cannot be changed after creation');
}

// Log the update
console.log('Updating pet:', this.name, 'by user:', this.updatedBy);

// Protect sensitive fields
protect('createdBy');
protect('updatedBy');